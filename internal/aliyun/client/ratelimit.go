/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const longThrottleLatency = 5 * time.Second

type ecsOperation uint8

const (
	ecsDescribeInstances ecsOperation = iota
	ecsDescribeNetworkInterfaces
	ecsCreateNetworkInterface
	ecsModifyNetworkInterfaceAttribute
	ecsTagResources
	ecsDescribeInstanceTypes
	ecsAttachNetworkInterface
	ecsDescribeInstanceAttribute
	ecsOperationCount
)

type ecsOperationConfig struct {
	name             string
	defaultPerMinute int
}

// Conservative per-minute defaults: one quarter of each ECS OpenAPI's documented quota.
var ecsOperationConfigs = [ecsOperationCount]ecsOperationConfig{
	ecsDescribeInstances:               {name: "DescribeInstances", defaultPerMinute: 250},
	ecsDescribeNetworkInterfaces:       {name: "DescribeNetworkInterfaces", defaultPerMinute: 200},
	ecsCreateNetworkInterface:          {name: "CreateNetworkInterface", defaultPerMinute: 125},
	ecsModifyNetworkInterfaceAttribute: {name: "ModifyNetworkInterfaceAttribute", defaultPerMinute: 125},
	ecsTagResources:                    {name: "TagResources", defaultPerMinute: 250},
	ecsDescribeInstanceTypes:           {name: "DescribeInstanceTypes", defaultPerMinute: 100},
	ecsAttachNetworkInterface:          {name: "AttachNetworkInterface", defaultPerMinute: 125},
	ecsDescribeInstanceAttribute:       {name: "DescribeInstanceAttribute", defaultPerMinute: 500},
}

type rateLimiter struct {
	store [ecsOperationCount]*rate.Limiter
}

func operationByName(name string) (ecsOperation, bool) {
	for operation, config := range ecsOperationConfigs {
		if config.name == name {
			return ecsOperation(operation), true
		}
	}
	return 0, false
}

func newRateLimiter(overrides map[string]int) (*rateLimiter, error) {
	for name, perMinute := range overrides {
		if _, ok := operationByName(name); !ok {
			return nil, fmt.Errorf("unsupported ECS OpenAPI rate limit %q", name)
		}
		if perMinute <= 0 {
			return nil, fmt.Errorf("ECS OpenAPI rate limit %q must be positive", name)
		}
	}

	r := &rateLimiter{}
	for operation, config := range ecsOperationConfigs {
		perMinute := config.defaultPerMinute
		if override, ok := overrides[config.name]; ok {
			perMinute = override
		}
		r.store[operation] = rate.NewLimiter(rate.Limit(float64(perMinute)/60), perMinute)
	}
	return r, nil
}

func (r *rateLimiter) wait(ctx context.Context, operation ecsOperation) error {
	if operation >= ecsOperationCount {
		return fmt.Errorf("unsupported ECS OpenAPI operation %d", operation)
	}
	start := time.Now()
	defer func() {
		took := time.Since(start)
		if took >= longThrottleLatency {
			logf.FromContext(ctx).Info(
				"client rate limit",
				"api", ecsOperationConfigs[operation].name,
				"took", took.Seconds(),
			)
		}
	}()
	return r.store[operation].Wait(ctx)
}
