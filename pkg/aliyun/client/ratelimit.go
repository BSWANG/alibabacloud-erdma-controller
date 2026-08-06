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
	"time"

	"golang.org/x/time/rate"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Copied from Terway pkg/aliyun/client/ratelimit.go at commit
// 89aadf8bbc19dbc63f51161b6453b634cfe9986e. Local differences are the
// omitted Terway metric and ERDMA API entries.

type LimitConfig map[string]Limit

type Limit struct {
	QPS   float64
	Burst int
}

var defaultLimit = map[string]int{
	"":                                  500,
	"AttachNetworkInterface":            500,
	"CreateNetworkInterface":            500,
	"DeleteNetworkInterface":            500,
	"DescribeInstanceAttribute":         2000,
	"DescribeInstances":                 1000,
	"DescribeNetworkInterfaces":         800,
	"DescribeNetworkInterfaceAttribute": 2000,
	"ModifyNetworkInterfaceAttribute":   500,
	"TagResources":                      1000,
	"DetachNetworkInterface":            400,
	"AssignPrivateIpAddresses":          400,
	"UnassignPrivateIpAddresses":        400,
	"AssignIpv6Addresses":               400,
	"UnassignIpv6Addresses":             400,
	"DescribeInstanceTypes":             400,
	"DescribeVSwitches":                 300,
	// eflo
	"AssignLeniPrivateIpAddress":               300,
	"AttachElasticNetworkInterface":            300,
	"DetachElasticNetworkInterface":            300,
	"ListElasticNetworkInterfaces":             100 * 60,
	"CreateElasticNetworkInterface":            20 * 60,
	"DeleteElasticNetworkInterface":            20 * 60,
	"CreateHighDensityElasticNetworkInterface": 15 * 60,
	"DeleteHighDensityElasticNetworkInterface": 300,
	"AttachHighDensityElasticNetworkInterface": 300,
	"DetachHighDensityElasticNetworkInterface": 300,
	"ListHighDensityElasticNetworkInterfaces":  100 * 60,
	"GetNodeInfoForPod":                        100 * 60,
}

const longThrottleLatency = 5 * time.Second

func FromMap(in map[string]int) LimitConfig {
	l := make(LimitConfig)
	for k, v := range in {
		l[k] = Limit{
			QPS:   float64(v) / 60,
			Burst: v,
		}
	}
	return l
}

type RateLimiter struct {
	store map[string]*rate.Limiter
}

func NewRateLimiter(cfg LimitConfig) *RateLimiter {
	r := &RateLimiter{
		store: make(map[string]*rate.Limiter),
	}
	for k, v := range defaultLimit {
		r.store[k] = rate.NewLimiter(rate.Limit(float64(v)/60), v)
	}
	for k, v := range cfg {
		r.store[k] = rate.NewLimiter(rate.Limit(v.QPS), v.Burst)
	}

	return r
}

func (r *RateLimiter) Wait(ctx context.Context, name string) error {
	start := time.Now()
	defer func() {
		took := time.Since(start)
		if took >= longThrottleLatency {
			l := logf.FromContext(ctx)
			l.Info("client rate limit", "api", name, "took", took.Seconds())
		}
	}()
	v, ok := r.store[name]
	if ok {
		return v.Wait(ctx)
	}
	return r.store[""].Wait(ctx)
}
