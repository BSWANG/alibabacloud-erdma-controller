package client

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/time/rate"
)

func TestNewRateLimiterConvertsPerMinuteAndKeepsManagedBurst(t *testing.T) {
	r, err := newRateLimiter(nil, nil)
	if err != nil {
		t.Fatalf("newRateLimiter() error = %v", err)
	}
	for operation, config := range ecsOperationConfigs {
		limiter := r.store[operation]
		wantQPS := float64(config.defaultRate.perMinute) / 60
		if float64(limiter.Limit()) != wantQPS {
			t.Errorf("%s QPS = %v, want %v", config.name, limiter.Limit(), wantQPS)
		}
		if limiter.Burst() != config.defaultRate.burst {
			t.Errorf("%s burst = %d, want %d", config.name, limiter.Burst(), config.defaultRate.burst)
		}
	}

	const perMinute = 750
	name := ecsOperationConfigs[ecsDescribeInstances].name
	r, err = newRateLimiter(map[string]int{name: perMinute}, map[string]int{name: 1})
	if err != nil {
		t.Fatalf("newRateLimiter() with override error = %v", err)
	}
	if got, want := float64(r.store[ecsDescribeInstances].Limit()), float64(perMinute)/60; got != want {
		t.Errorf("DescribeInstances QPS = %v, want %v", got, want)
	}
	if got := r.store[ecsDescribeInstances].Burst(); got != 1 {
		t.Errorf("DescribeInstances burst = %d, want override 1", got)
	}
}
func TestModifyNetworkInterfaceAttributeDefaultRate(t *testing.T) {
	if got := ecsOperationConfigs[ecsModifyNetworkInterfaceAttribute].defaultRate.perMinute; got != 500 {
		t.Fatalf("ModifyNetworkInterfaceAttribute requests/minute = %d, want 500", got)
	}
}

func TestNewRateLimiterRejectsInvalidOverrides(t *testing.T) {
	describeInstances := ecsOperationConfigs[ecsDescribeInstances].name
	tests := []struct {
		rates  map[string]int
		bursts map[string]int
	}{
		{rates: map[string]int{"UnusedOpenAPI": 1}},
		{rates: map[string]int{describeInstances: 0}},
		{rates: map[string]int{describeInstances: -1}},
		{bursts: map[string]int{"UnusedOpenAPI": 1}},
		{bursts: map[string]int{describeInstances: 0}},
		{bursts: map[string]int{describeInstances: -1}},
	}
	for _, tt := range tests {
		if _, err := newRateLimiter(tt.rates, tt.bursts); err == nil {
			t.Errorf("newRateLimiter(%v, %v) succeeded", tt.rates, tt.bursts)
		}
	}
}

func TestRateLimiterWaitHonorsContext(t *testing.T) {
	describeInstances := ecsOperationConfigs[ecsDescribeInstances].name
	r, err := newRateLimiter(map[string]int{
		describeInstances: 1,
	}, nil)
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
	r, err := newRateLimiter(nil, nil)
	if err != nil {
		t.Fatalf("newRateLimiter() error = %v", err)
	}
	if err := r.wait(context.Background(), ecsOperationCount); err == nil {
		t.Fatal("wait() accepted an unknown ECS operation")
	}
}
