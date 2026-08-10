package client

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/time/rate"
)

func TestNewRateLimiterConvertsPerMinute(t *testing.T) {
	r, err := newRateLimiter(nil)
	if err != nil {
		t.Fatalf("newRateLimiter() error = %v", err)
	}
	for operation, config := range ecsOperationConfigs {
		limiter := r.store[operation]
		wantQPS := float64(config.defaultPerMinute) / 60
		if float64(limiter.Limit()) != wantQPS {
			t.Errorf("%s QPS = %v, want %v", config.name, limiter.Limit(), wantQPS)
		}
	}

	const perMinute = 750
	name := ecsOperationConfigs[ecsDescribeInstances].name
	r, err = newRateLimiter(map[string]int{name: perMinute})
	if err != nil {
		t.Fatalf("newRateLimiter() with override error = %v", err)
	}
	if got, want := float64(r.store[ecsDescribeInstances].Limit()), float64(perMinute)/60; got != want {
		t.Errorf("DescribeInstances QPS = %v, want %v", got, want)
	}
}

func TestModifyNetworkInterfaceAttributeDefaultRate(t *testing.T) {
	if got := ecsOperationConfigs[ecsModifyNetworkInterfaceAttribute].defaultPerMinute; got != 500 {
		t.Fatalf("ModifyNetworkInterfaceAttribute requests/minute = %d, want 500", got)
	}
}

func TestNewRateLimiterRejectsInvalidOverrides(t *testing.T) {
	describeInstances := ecsOperationConfigs[ecsDescribeInstances].name
	tests := []map[string]int{
		{"UnusedOpenAPI": 1},
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
	r, err := newRateLimiter(map[string]int{
		describeInstances: 1,
	})
	if err != nil {
		t.Fatalf("newRateLimiter() error = %v", err)
	}
	r.store[ecsDescribeInstances] = rate.NewLimiter(0.01, 1)
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
