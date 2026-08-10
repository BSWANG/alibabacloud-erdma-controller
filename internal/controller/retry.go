package controller

import (
	"errors"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ecsThrottlingError interface {
	error
	Operation() string
	Code() string
	StatusCode() int
	RetryAfter() time.Duration
}

func requeueOnECSThrottling(err error, logger logr.Logger) (ctrl.Result, error) {
	var throttlingErr ecsThrottlingError
	if !errors.As(err, &throttlingErr) {
		return ctrl.Result{}, err
	}
	logger.Info(
		"ECS API throttled; requeueing without blocking a reconcile worker",
		"api", throttlingErr.Operation(),
		"code", throttlingErr.Code(),
		"statusCode", throttlingErr.StatusCode(),
		"retryAfter", throttlingErr.RetryAfter(),
	)
	return ctrl.Result{RequeueAfter: throttlingErr.RetryAfter()}, nil
}
