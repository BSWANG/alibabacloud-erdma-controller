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
	"errors"
	"github.com/alibabacloud-go/tea/tea"
	"testing"

	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
	"golang.org/x/time/rate"
)

type fakeECSAPI struct {
	calls                map[ecsOperation]int
	describeInstancesErr error
}

func (f *fakeECSAPI) DescribeInstances(*ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
	f.calls[ecsDescribeInstances]++
	if f.describeInstancesErr != nil {
		return nil, f.describeInstancesErr
	}
	return &ecs.DescribeInstancesResponse{}, nil
}

func (f *fakeECSAPI) DescribeNetworkInterfaces(*ecs.DescribeNetworkInterfacesRequest) (*ecs.DescribeNetworkInterfacesResponse, error) {
	f.calls[ecsDescribeNetworkInterfaces]++
	return &ecs.DescribeNetworkInterfacesResponse{}, nil
}

func (f *fakeECSAPI) CreateNetworkInterface(*ecs.CreateNetworkInterfaceRequest) (*ecs.CreateNetworkInterfaceResponse, error) {
	f.calls[ecsCreateNetworkInterface]++
	return &ecs.CreateNetworkInterfaceResponse{}, nil
}

func (f *fakeECSAPI) ModifyNetworkInterfaceAttribute(*ecs.ModifyNetworkInterfaceAttributeRequest) (*ecs.ModifyNetworkInterfaceAttributeResponse, error) {
	f.calls[ecsModifyNetworkInterfaceAttribute]++
	return &ecs.ModifyNetworkInterfaceAttributeResponse{}, nil
}

func (f *fakeECSAPI) TagResources(*ecs.TagResourcesRequest) (*ecs.TagResourcesResponse, error) {
	f.calls[ecsTagResources]++
	return &ecs.TagResourcesResponse{}, nil
}

func (f *fakeECSAPI) DescribeInstanceTypes(*ecs.DescribeInstanceTypesRequest) (*ecs.DescribeInstanceTypesResponse, error) {
	f.calls[ecsDescribeInstanceTypes]++
	return &ecs.DescribeInstanceTypesResponse{}, nil
}

func (f *fakeECSAPI) AttachNetworkInterface(*ecs.AttachNetworkInterfaceRequest) (*ecs.AttachNetworkInterfaceResponse, error) {
	f.calls[ecsAttachNetworkInterface]++
	return &ecs.AttachNetworkInterfaceResponse{}, nil
}

func (f *fakeECSAPI) DescribeInstanceAttribute(*ecs.DescribeInstanceAttributeRequest) (*ecs.DescribeInstanceAttributeResponse, error) {
	f.calls[ecsDescribeInstanceAttribute]++
	return &ecs.DescribeInstanceAttributeResponse{}, nil
}

func TestECSServiceRateLimitsBeforeCallingSDK(t *testing.T) {
	tests := []struct {
		operation ecsOperation
		call      func(context.Context, *ECSService) error
	}{
		{operation: ecsDescribeInstances, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.DescribeInstances(ctx, &ecs.DescribeInstancesRequest{})
			return err
		}},
		{operation: ecsDescribeNetworkInterfaces, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.DescribeNetworkInterfaces(ctx, &ecs.DescribeNetworkInterfacesRequest{})
			return err
		}},
		{operation: ecsCreateNetworkInterface, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.CreateNetworkInterface(ctx, &ecs.CreateNetworkInterfaceRequest{})
			return err
		}},
		{operation: ecsModifyNetworkInterfaceAttribute, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.ModifyNetworkInterfaceAttribute(ctx, &ecs.ModifyNetworkInterfaceAttributeRequest{})
			return err
		}},
		{operation: ecsTagResources, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.TagResources(ctx, &ecs.TagResourcesRequest{})
			return err
		}},
		{operation: ecsDescribeInstanceTypes, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.DescribeInstanceTypes(ctx, &ecs.DescribeInstanceTypesRequest{})
			return err
		}},
		{operation: ecsAttachNetworkInterface, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.AttachNetworkInterface(ctx, &ecs.AttachNetworkInterfaceRequest{})
			return err
		}},
		{operation: ecsDescribeInstanceAttribute, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.DescribeInstanceAttribute(ctx, &ecs.DescribeInstanceAttributeRequest{})
			return err
		}},
	}

	for _, tt := range tests {
		name := ecsOperationConfigs[tt.operation].name
		t.Run(name, func(t *testing.T) {
			rawClient := &fakeECSAPI{calls: map[ecsOperation]int{}}
			client, err := NewECSService(rawClient, map[string]int{name: 1}, nil)
			if err != nil {
				t.Fatalf("NewECSService() error = %v", err)
			}
			client.rateLimiter.store[tt.operation] = rate.NewLimiter(0.01, 1)

			if err := tt.call(context.Background(), client); err != nil {
				t.Fatalf("first call error = %v", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := tt.call(ctx, client); !errors.Is(err, context.Canceled) {
				t.Fatalf("second call error = %v, want context.Canceled", err)
			}
			if rawClient.calls[tt.operation] != 1 {
				t.Fatalf("SDK calls = %d, want 1", rawClient.calls[tt.operation])
			}
		})
	}
}

func TestECSServiceClassifiesRemoteThrottlingWithoutHTTP429(t *testing.T) {
	sdkErr := &tea.SDKError{
		StatusCode: tea.Int(400),
		Code:       tea.String("Throttling"),
		Message:    tea.String("request was denied due to request throttling"),
	}
	rawClient := &fakeECSAPI{
		calls:                map[ecsOperation]int{},
		describeInstancesErr: sdkErr,
	}
	client, err := NewECSService(rawClient, nil, nil)
	if err != nil {
		t.Fatalf("NewECSService() error = %v", err)
	}

	_, err = client.DescribeInstances(context.Background(), &ecs.DescribeInstancesRequest{})
	var throttlingErr *ThrottlingError
	if !errors.As(err, &throttlingErr) {
		t.Fatalf("DescribeInstances() error = %T %v, want *ThrottlingError", err, err)
	}
	if throttlingErr.Operation() != "DescribeInstances" || throttlingErr.Code() != "Throttling" || throttlingErr.StatusCode() != 400 {
		t.Fatalf("ThrottlingError = api %q code %q status %d", throttlingErr.Operation(), throttlingErr.Code(), throttlingErr.StatusCode())
	}
	if throttlingErr.RetryAfter() < minThrottlingRetryDelay || throttlingErr.RetryAfter() > maxThrottlingRetryDelay {
		t.Fatalf("RetryAfter = %v, want [%v,%v]", throttlingErr.RetryAfter(), minThrottlingRetryDelay, maxThrottlingRetryDelay)
	}
	if !errors.Is(err, sdkErr) {
		t.Fatal("ThrottlingError does not preserve the SDK error")
	}
}

func TestWrapECSAPIErrorClassifiesHTTP429(t *testing.T) {
	sdkErr := &tea.SDKError{
		StatusCode: tea.Int(429),
		Code:       tea.String("UnknownLimitCode"),
	}
	err := wrapECSAPIError(ecsAttachNetworkInterface, sdkErr)
	var throttlingErr *ThrottlingError
	if !errors.As(err, &throttlingErr) {
		t.Fatalf("wrapECSAPIError() error = %T %v, want *ThrottlingError", err, err)
	}
	if throttlingErr.Operation() != "AttachNetworkInterface" || throttlingErr.StatusCode() != 429 {
		t.Fatalf("ThrottlingError = api %q status %d", throttlingErr.Operation(), throttlingErr.StatusCode())
	}
}

func TestECSServiceDoesNotMisclassifyNonThrottlingSDKError(t *testing.T) {
	sdkErr := &tea.SDKError{StatusCode: tea.Int(400), Code: tea.String("InvalidParameter")}
	rawClient := &fakeECSAPI{
		calls:                map[ecsOperation]int{},
		describeInstancesErr: sdkErr,
	}
	client, err := NewECSService(rawClient, nil, nil)
	if err != nil {
		t.Fatalf("NewECSService() error = %v", err)
	}
	_, err = client.DescribeInstances(context.Background(), &ecs.DescribeInstancesRequest{})
	var throttlingErr *ThrottlingError
	if errors.As(err, &throttlingErr) {
		t.Fatalf("DescribeInstances() error = %#v, unexpectedly classified as throttling", err)
	}
	if !errors.Is(err, sdkErr) {
		t.Fatalf("DescribeInstances() error = %v, want original SDK error", err)
	}
}
