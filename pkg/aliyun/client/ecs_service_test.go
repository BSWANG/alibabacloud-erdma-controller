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
	"testing"

	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
)

type fakeECSAPI struct {
	calls map[string]int
}

func (f *fakeECSAPI) DescribeInstances(*ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
	f.calls[APIDescribeInstances]++
	return &ecs.DescribeInstancesResponse{}, nil
}

func (f *fakeECSAPI) DescribeNetworkInterfaces(*ecs.DescribeNetworkInterfacesRequest) (*ecs.DescribeNetworkInterfacesResponse, error) {
	f.calls[APIDescribeNetworkInterfaces]++
	return &ecs.DescribeNetworkInterfacesResponse{}, nil
}

func (f *fakeECSAPI) CreateNetworkInterface(*ecs.CreateNetworkInterfaceRequest) (*ecs.CreateNetworkInterfaceResponse, error) {
	f.calls[APICreateNetworkInterface]++
	return &ecs.CreateNetworkInterfaceResponse{}, nil
}

func (f *fakeECSAPI) ModifyNetworkInterfaceAttribute(*ecs.ModifyNetworkInterfaceAttributeRequest) (*ecs.ModifyNetworkInterfaceAttributeResponse, error) {
	f.calls[APIModifyNetworkInterfaceAttribute]++
	return &ecs.ModifyNetworkInterfaceAttributeResponse{}, nil
}

func (f *fakeECSAPI) TagResources(*ecs.TagResourcesRequest) (*ecs.TagResourcesResponse, error) {
	f.calls[APITagResources]++
	return &ecs.TagResourcesResponse{}, nil
}

func (f *fakeECSAPI) DescribeInstanceTypes(*ecs.DescribeInstanceTypesRequest) (*ecs.DescribeInstanceTypesResponse, error) {
	f.calls[APIDescribeInstanceTypes]++
	return &ecs.DescribeInstanceTypesResponse{}, nil
}

func (f *fakeECSAPI) AttachNetworkInterface(*ecs.AttachNetworkInterfaceRequest) (*ecs.AttachNetworkInterfaceResponse, error) {
	f.calls[APIAttachNetworkInterface]++
	return &ecs.AttachNetworkInterfaceResponse{}, nil
}

func (f *fakeECSAPI) DescribeInstanceAttribute(*ecs.DescribeInstanceAttributeRequest) (*ecs.DescribeInstanceAttributeResponse, error) {
	f.calls[APIDescribeInstanceAttribute]++
	return &ecs.DescribeInstanceAttributeResponse{}, nil
}

func TestECSServiceRateLimitsBeforeCallingSDK(t *testing.T) {
	tests := []struct {
		api  string
		call func(context.Context, *ECSService) error
	}{
		{api: APIDescribeInstances, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.DescribeInstances(ctx, &ecs.DescribeInstancesRequest{})
			return err
		}},
		{api: APIDescribeNetworkInterfaces, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.DescribeNetworkInterfaces(ctx, &ecs.DescribeNetworkInterfacesRequest{})
			return err
		}},
		{api: APICreateNetworkInterface, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.CreateNetworkInterface(ctx, &ecs.CreateNetworkInterfaceRequest{})
			return err
		}},
		{api: APIModifyNetworkInterfaceAttribute, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.ModifyNetworkInterfaceAttribute(ctx, &ecs.ModifyNetworkInterfaceAttributeRequest{})
			return err
		}},
		{api: APITagResources, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.TagResources(ctx, &ecs.TagResourcesRequest{})
			return err
		}},
		{api: APIDescribeInstanceTypes, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.DescribeInstanceTypes(ctx, &ecs.DescribeInstanceTypesRequest{})
			return err
		}},
		{api: APIAttachNetworkInterface, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.AttachNetworkInterface(ctx, &ecs.AttachNetworkInterfaceRequest{})
			return err
		}},
		{api: APIDescribeInstanceAttribute, call: func(ctx context.Context, client *ECSService) error {
			_, err := client.DescribeInstanceAttribute(ctx, &ecs.DescribeInstanceAttributeRequest{})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.api, func(t *testing.T) {
			rawClient := &fakeECSAPI{calls: map[string]int{}}
			limiter := NewRateLimiter(LimitConfig{
				tt.api: {QPS: 1, Burst: 1},
			})
			client := NewECSService(rawClient, limiter)

			if err := tt.call(context.Background(), client); err != nil {
				t.Fatalf("first call error = %v", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := tt.call(ctx, client); !errors.Is(err, context.Canceled) {
				t.Fatalf("second call error = %v, want context.Canceled", err)
			}
			if rawClient.calls[tt.api] != 1 {
				t.Fatalf("SDK calls = %d, want 1", rawClient.calls[tt.api])
			}
		})
	}
}
