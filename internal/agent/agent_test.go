package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/AliyunContainerService/alibabacloud-erdma-controller/internal/drivers"
	"github.com/AliyunContainerService/alibabacloud-erdma-controller/internal/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

type retryTestDriver struct {
	failures []error
	calls    int
}

func (d *retryTestDriver) Install() error { return nil }

func (d *retryTestDriver) ProbeDevice(*types.ERI) (*types.ERdmaDeviceInfo, error) {
	d.calls++
	if d.calls <= len(d.failures) {
		return nil, d.failures[d.calls-1]
	}
	return &types.ERdmaDeviceInfo{Name: "erdma_0"}, nil
}

func (d *retryTestDriver) Name() string { return "retry-test" }

func (d *retryTestDriver) SetERdmaInstallerVersion(string) {}

func immediateBackoff(steps int) wait.Backoff {
	return wait.Backoff{Steps: steps}
}

func TestProbeDeviceWithRetryEventuallySucceeds(t *testing.T) {
	linkNotReady := fmt.Errorf("get erdma link failed: %w", drivers.ErrERdmaLinkNotFound)
	driver := &retryTestDriver{failures: []error{linkNotReady, linkNotReady}}

	device, err := probeDeviceWithRetry(context.Background(), driver, &types.ERI{ID: "eni-test"}, immediateBackoff(3))
	if err != nil {
		t.Fatalf("probeDeviceWithRetry() error = %v", err)
	}
	if device == nil || device.Name != "erdma_0" {
		t.Fatalf("probeDeviceWithRetry() device = %#v", device)
	}
	if driver.calls != 3 {
		t.Fatalf("ProbeDevice() calls = %d, want 3", driver.calls)
	}
}

func TestProbeDeviceWithRetryStopsOnPermanentError(t *testing.T) {
	permanentErr := errors.New("invalid device configuration")
	driver := &retryTestDriver{failures: []error{permanentErr}}

	_, err := probeDeviceWithRetry(context.Background(), driver, &types.ERI{ID: "eni-test"}, immediateBackoff(3))
	if !errors.Is(err, permanentErr) {
		t.Fatalf("probeDeviceWithRetry() error = %v, want %v", err, permanentErr)
	}
	if driver.calls != 1 {
		t.Fatalf("ProbeDevice() calls = %d, want 1", driver.calls)
	}
}

func TestProbeDeviceWithRetryReturnsLastTransientError(t *testing.T) {
	linkNotReady := fmt.Errorf("get erdma link failed: %w", drivers.ErrERdmaLinkNotFound)
	driver := &retryTestDriver{failures: []error{linkNotReady, linkNotReady, linkNotReady}}

	_, err := probeDeviceWithRetry(context.Background(), driver, &types.ERI{ID: "eni-test"}, immediateBackoff(3))
	if !errors.Is(err, drivers.ErrERdmaLinkNotFound) {
		t.Fatalf("probeDeviceWithRetry() error = %v, want ErrERdmaLinkNotFound", err)
	}
	if driver.calls != 3 {
		t.Fatalf("ProbeDevice() calls = %d, want 3", driver.calls)
	}
}

func TestProbeDeviceWithRetryHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	driver := &retryTestDriver{}

	_, err := probeDeviceWithRetry(ctx, driver, &types.ERI{ID: "eni-test"}, immediateBackoff(3))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("probeDeviceWithRetry() error = %v, want context.Canceled", err)
	}
	if driver.calls != 0 {
		t.Fatalf("ProbeDevice() calls = %d, want 0", driver.calls)
	}
}
