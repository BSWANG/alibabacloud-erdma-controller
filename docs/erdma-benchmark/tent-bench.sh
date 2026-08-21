#!/usr/bin/env bash
# Mooncake Transfer Engine (classic vs TENT) over eRDMA 跨节点压测驱动脚本
#
# 前置: 已 kubectl apply -f tent-pods.yaml 且两个 pod Ready.
# 作用: 自动发现 pod -> A 起 tebench target -> B 依次以 TENT / classic backend
#       做跨节点 DRAM 传输压测, 并附 perftest verbs 层带宽对照.
#
# 用法:
#   ./tent-bench.sh                 # 默认 8 线程大块
#   THREADS=4 DURATION=10 ./tent-bench.sh
#   NS=default ./tent-bench.sh
set -euo pipefail

NS="${NS:-default}"
THREADS="${THREADS:-8}"
DURATION="${DURATION:-5}"

k() { kubectl -n "$NS" "$@"; }

PODS=()
while IFS= read -r line; do PODS+=("$line"); done < <(k get pods -l app=tent-test -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')
[[ ${#PODS[@]} -ge 2 ]] || { echo "找不到 2 个 app=tent-test 的 pod, 请先 apply tent-pods.yaml"; exit 1; }
A="${PODS[0]}"; B="${PODS[1]}"
IPA=$(k get pod "$A" -o jsonpath='{.status.podIP}')
IPB=$(k get pod "$B" -o jsonpath='{.status.podIP}')
echo "== pods: $A($IPA)  $B($IPB) =="

# classic backend p2p 模式用 hostname 作段名, initiator 需能解析 target 的 hostname
for p in "$A" "$B"; do
  k exec "$p" -- bash -c "grep -q ' $A\$' /etc/hosts || echo '$IPA $A' >> /etc/hosts; grep -q ' $B\$' /etc/hosts || echo '$IPB $B' >> /etc/hosts"
done

# --- 注入 tebench 二进制 (本目录 bin/ 下, 源码编译产物) + 运行期库 ---
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [[ -x "$SCRIPT_DIR/bin/tebench" ]]; then
  for p in "$A" "$B"; do
    if ! k exec "$p" -- test -x /root/tebench; then
      echo "== 注入 tebench 到 $p =="
      k cp "$SCRIPT_DIR/bin/tebench" "$p:/root/tebench"
      [[ -f "$SCRIPT_DIR/bin/transfer_engine_bench" ]] && k cp "$SCRIPT_DIR/bin/transfer_engine_bench" "$p:/root/transfer_engine_bench"
      k exec "$p" -- chmod +x /root/tebench /root/transfer_engine_bench
    fi
  done
else
  echo "WARN: $SCRIPT_DIR/bin/tebench 不存在, 仅当 pod 内已有 tebench 时可继续"
fi
for p in "$A" "$B"; do
  k exec "$p" -- bash -c 'export DEBIAN_FRONTEND=noninteractive; apt-get update -qq >/dev/null 2>&1; apt-get install -y -qq liburing2 libnuma1 libcurl4 libhiredis0.14 libgflags2.2 libgoogle-glog0v5 libjsoncpp25 libunwind8 >/dev/null 2>&1 || true'
  k exec "$p" -- sh -c 'ldd /root/tebench | grep -q "not found" && echo "WARN: $p 仍有缺失依赖: $(ldd /root/tebench | grep "not found")" || true'
done
TEBENCH="${TEBENCH:-/root/tebench}"

echo "== verbs 设备 =="
k exec "$A" -- ibv_devinfo -d erdma_0 | head -12 || true
GID_IDX=$(k exec "$B" -- sh -c 'for i in $(seq 0 15); do g=$(cat /sys/class/infiniband/erdma_0/ports/1/gids/$i 2>/dev/null); t=$(cat /sys/class/infiniband/erdma_0/ports/1/gid_attrs/types/$i 2>/dev/null); case "$g" in *ffff:*:*) [ "$t" = "RoCE v2" ] && { echo $i; exit 0; };; esac; done; echo 0')
GID_IDX="${GID_IDX:-0}"
echo "== RoCEv2 IPv4 GID index: $GID_IDX =="

# --- perftest 对照 (verbs 层单网卡带宽) ---
echo; echo "########## perftest ib_write_bw (baseline) ##########"
k exec "$A" -- sh -c "ib_write_bw -d erdma_0 -x $GID_IDX -F --report_gbits -D 10 >/tmp/pft.log 2>&1" &
sleep 2
k exec "$B" -- sh -c "ib_write_bw -d erdma_0 -x $GID_IDX -F --report_gbits -D 10 $IPA" | tail -4 || true
wait || true
k exec "$A" -- tail -4 /tmp/pft.log 2>/dev/null || true

# --- tebench: A 起 target, 返回 target_seg_name ---
start_target() { # $1=backend -> 输出 target_seg_name
  k exec "$A" -- sh -c "pkill -x tebench 2>/dev/null; rm -f /tmp/target.log; \
    nohup $TEBENCH --backend=$1 --seg_type=DRAM --metadata_type=p2p --seg_name=$IPA \
      --total_buffer_size=4294967296 >/tmp/target.log 2>&1 &"
  for _ in $(seq 1 30); do
    k exec "$A" -- grep -q 'target_seg_name=' /tmp/target.log 2>/dev/null && break
    sleep 1
  done
  local seg
  seg=$(k exec "$A" -- grep -oE 'target_seg_name=[^ ]+' /tmp/target.log | head -1 | cut -d= -f2-)
  if [[ -z "$seg" ]]; then
    echo "ERROR: target 未就绪" >&2
    k exec "$A" -- tail -20 /tmp/target.log >&2
    return 1
  fi
  echo "$seg"
}

run_initiator() { # $1=backend $2=target_seg  $3...=额外 flags
  local backend="$1" seg="$2"; shift 2
  k exec "$B" -- bash -c "$TEBENCH --backend=$backend --seg_type=DRAM --metadata_type=p2p \
      --target_seg_name='$seg' --duration=$DURATION --total_buffer_size=4294967296 $* 2>&1" \
    | grep -vE '^I[0-9]{4}|^W[0-9]{4}|^E[0-9]{4} '
}

for BACKEND in tent classic; do
  echo; echo "########## tebench --backend=$BACKEND (1-thread sweep) ##########"
  TARGET_SEG=$(start_target "$BACKEND")
  echo "== target_seg_name=$TARGET_SEG =="
  run_initiator "$BACKEND" "$TARGET_SEG" --start_num_threads=1 --max_num_threads=1
  echo; echo "########## tebench --backend=$BACKEND ($THREADS-thread, 大块) ##########"
  run_initiator "$BACKEND" "$TARGET_SEG" \
      --start_num_threads="$THREADS" --max_num_threads="$THREADS" \
      --start_block_size=1048576 --max_block_size=67108864 \
      --start_batch_size=1 --max_batch_size=4
  echo; echo "########## tebench --backend=$BACKEND (consistency) ##########"
  run_initiator "$BACKEND" "$TARGET_SEG" --start_num_threads=2 --max_num_threads=2 \
      --start_block_size=65536 --max_block_size=65536 --duration=3 --op_type=write_seed
  run_initiator "$BACKEND" "$TARGET_SEG" --start_num_threads=2 --max_num_threads=2 \
      --start_block_size=65536 --max_block_size=65536 --duration=3 --op_type=read_verify
done

k exec "$A" -- sh -c "pkill -x tebench" 2>/dev/null || true
echo; echo "== done =="
