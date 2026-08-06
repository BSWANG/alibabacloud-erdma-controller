package client

import (
	"context"
	"errors"
	"testing"
)

func TestNewRateLimiterUsesDefaultsAndOverrides(t *testing.T) {
	expectedBursts := [ecsOperationCount]int{
		ecsDescribeInstances:               250,
		ecsDescribeNetworkInterfaces:       200,
		ecsCreateNetworkInterface:          125,
		ecsModifyNetworkInterfaceAttribute: 125,
		ecsTagResources:                    250,
		ecsDescribeInstanceTypes:           100,
		ecsAttachNetworkInterface:          125,
		ecsDescribeInstanceAttribute:       500,
	}

	r, err := newRateLimiter(nil)
	if err != nil {
		t.Fatalf("newRateLimiter() error = %v", err)
	}
	for operation, burst := range expectedBursts {
		if r.store[operation].Burst() != burst {
			t.Errorf("%s burst = %d, want %d", ecsOperationConfigs[operation].name, r.store[operation].Burst(), burst)
		}
	}

	const override = 150
	r, err = newRateLimiter(map[string]int{
		ecsOperationConfigs[ecsDescribeInstances].name: override,
	})
	if err != nil {
		t.Fatalf("newRateLimiter() with override error = %v", err)
	}
	if r.store[ecsDescribeInstances].Burst() != override {
		t.Errorf("DescribeInstances override burst = %d, want %d", r.store[ecsDescribeInstances].Burst(), override)
	}
}

func TestNewRateLimiterRejectsInvalidOverrides(t *testing.T) {
	describeInstances := ecsOperationConfigs[ecsDescribeInstances].name
	tests := []map[string]int{
		{"UnusedOpenAPI": 900},
		{describeInstances: 0},
		{describeInstances: -1},
	}
	for _, overrides := range tests {
		if _, err := newRateLimiter(overrides); err == nil {
			t.Errorf("newRateLimiter(%v) succeeded", overrides)
		}
	}
}

func TestRateLimiterWaitHonorsContext(t *testing.T) {
	describeInstances := ecsOperationConfigs[ecsDescribeInstances].name
	r, err := newRateLimiter(map[string]int{describeInstances: 1})
	if err != nil {
		t.Fatalf("newRateLimiter() error = %v", err)
	}
	if err := r.wait(context.Background(), ecsDescribeInstances); err != nil {
		t.Fatalf("first wait() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.wait(ctx, ecsDescribeInstances); !errors.Is(err, context.Canceled) {
		t.Fatalf("second wait() error = %v, want context.Canceled", err)
	}
}

func TestRateLimiterWaitRejectsUnknownOperation(t *testing.T) {
	r, err := newRateLimiter(nil)
	if err != nil {
		t.Fatalf("newRateLimiter() error = %v", err)
	}
	if err := r.wait(context.Background(), ecsOperationCount); err == nil {
		t.Fatal("wait() accepted an unknown ECS operation")
	}
}
