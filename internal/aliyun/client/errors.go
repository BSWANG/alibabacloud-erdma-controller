package client

import (
	"errors"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
)

const (
	minThrottlingRetryDelay = 5 * time.Second
	maxThrottlingRetryDelay = 15 * time.Second
)

// ThrottlingError carries a retry delay without discarding the SDK error.
type ThrottlingError struct {
	operation  string
	code       string
	statusCode int
	retryAfter time.Duration
	underlying error
}

func (e *ThrottlingError) Error() string {
	return e.underlying.Error()
}

func (e *ThrottlingError) Unwrap() error {
	return e.underlying
}

func (e *ThrottlingError) Operation() string {
	return e.operation
}

func (e *ThrottlingError) Code() string {
	return e.code
}

func (e *ThrottlingError) StatusCode() int {
	return e.statusCode
}

func (e *ThrottlingError) RetryAfter() time.Duration {
	return e.retryAfter
}

func IsThrottling(err error) bool {
	var throttlingErr *ThrottlingError
	return errors.As(err, &throttlingErr)
}

func wrapECSAPIError(operation ecsOperation, err error) error {
	if err == nil {
		return nil
	}
	var sdkErr *tea.SDKError
	if !errors.As(err, &sdkErr) {
		return err
	}
	code := tea.StringValue(sdkErr.Code)
	statusCode := tea.IntValue(sdkErr.StatusCode)
	if statusCode != 429 && code != "Throttling" && !strings.HasPrefix(code, "Throttling.") && code != "RequestLimitExceeded" {
		return err
	}
	jitterWindow := maxThrottlingRetryDelay - minThrottlingRetryDelay
	return &ThrottlingError{
		operation:  ecsOperationConfigs[operation].name,
		code:       code,
		statusCode: statusCode,
		retryAfter: minThrottlingRetryDelay + time.Duration(rand.Int64N(int64(jitterWindow)+1)),
		underlying: err,
	}
}
