package controller

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

type fakeECSThrottlingError struct {
	delay time.Duration
}

func (e *fakeECSThrottlingError) Error() string             { return "throttled" }
func (e *fakeECSThrottlingError) Operation() string         { return "DescribeInstances" }
func (e *fakeECSThrottlingError) Code() string              { return "Throttling" }
func (e *fakeECSThrottlingError) StatusCode() int           { return 400 }
func (e *fakeECSThrottlingError) RetryAfter() time.Duration { return e.delay }

func TestRequeueOnECSThrottlingReleasesWorker(t *testing.T) {
	throttlingErr := &fakeECSThrottlingError{delay: 7 * time.Second}
	result, err := requeueOnECSThrottling(fmt.Errorf("select ERIs: %w", throttlingErr), logr.Discard())
	if err != nil {
		t.Fatalf("requeueOnECSThrottling() error = %v", err)
	}
	if result.RequeueAfter != throttlingErr.delay {
		t.Fatalf("RequeueAfter = %v, want %v", result.RequeueAfter, throttlingErr.delay)
	}
}

func TestRequeueOnECSThrottlingPreservesOtherErrors(t *testing.T) {
	original := errors.New("invalid parameter")
	result, err := requeueOnECSThrottling(original, logr.Discard())
	if result.Requeue || result.RequeueAfter != 0 {
		t.Fatalf("result = %#v, want zero", result)
	}
	if !errors.Is(err, original) {
		t.Fatalf("error = %v, want original", err)
	}
}
