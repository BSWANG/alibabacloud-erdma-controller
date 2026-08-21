# Mooncake Transfer Engine + TENT 压测镜像（源码编译，USE_TENT=ON）
#
#   docker build -t registry.cn-hangzhou.aliyuncs.com/wangbs/erdma:mooncake-tent .
#
# CPU-only（DRAM 段传输）：USE_CUDA=OFF。GPU 显存传输需改 -DUSE_CUDA=ON 并用 CUDA 基础镜像。
FROM ubuntu:22.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
        git ca-certificates cmake ninja-build g++ make python3 python3-pip \
        pkg-config curl libssl-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
RUN git clone --depth 1 --recurse-submodules https://github.com/kvcache-ai/Mooncake.git
WORKDIR /workspace/Mooncake
RUN bash dependencies.sh -y

RUN mkdir -p build && cd build && \
    cmake -G Ninja .. \
        -DCMAKE_BUILD_TYPE=Release \
        -DUSE_TENT=ON \
        -DUSE_CUDA=OFF \
        -DWITH_STORE=OFF \
        -DWITH_STORE_RUST=OFF \
        -DWITH_EP=OFF \
        -DBUILD_UNIT_TESTS=OFF \
        -DBUILD_BENCHMARK=ON \
        -DBUILD_EXAMPLES=ON \
        -DUSE_HTTP=ON && \
    ninja tebench transfer_engine_bench

FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
        libibverbs1 ibverbs-providers librdmacm1 libnuma1 liburing2 \
        libcurl4 libgoogle-glog0 libgflags2.2 \
        perftest ibverbs-utils rdma-core \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /workspace/Mooncake/build/mooncake-transfer-engine/benchmark/tebench /usr/local/bin/
COPY --from=builder /workspace/Mooncake/build/mooncake-transfer-engine/example/transfer_engine_bench /usr/local/bin/

WORKDIR /root
CMD ["sleep", "infinity"]
