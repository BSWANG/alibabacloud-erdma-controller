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

	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
)

type ecsAPI interface {
	DescribeInstances(*ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error)
	DescribeNetworkInterfaces(*ecs.DescribeNetworkInterfacesRequest) (*ecs.DescribeNetworkInterfacesResponse, error)
	CreateNetworkInterface(*ecs.CreateNetworkInterfaceRequest) (*ecs.CreateNetworkInterfaceResponse, error)
	ModifyNetworkInterfaceAttribute(*ecs.ModifyNetworkInterfaceAttributeRequest) (*ecs.ModifyNetworkInterfaceAttributeResponse, error)
	TagResources(*ecs.TagResourcesRequest) (*ecs.TagResourcesResponse, error)
	DescribeInstanceTypes(*ecs.DescribeInstanceTypesRequest) (*ecs.DescribeInstanceTypesResponse, error)
	AttachNetworkInterface(*ecs.AttachNetworkInterfaceRequest) (*ecs.AttachNetworkInterfaceResponse, error)
	DescribeInstanceAttribute(*ecs.DescribeInstanceAttributeRequest) (*ecs.DescribeInstanceAttributeResponse, error)
}

// ECSService rate limits ECS API calls.
type ECSService struct {
	client      ecsAPI
	rateLimiter *rateLimiter
}

func NewECSService(client ecsAPI, rateOverrides map[string]int) (*ECSService, error) {
	limiter, err := newRateLimiter(rateOverrides)
	if err != nil {
		return nil, err
	}
	return &ECSService{
		client:      client,
		rateLimiter: limiter,
	}, nil
}

func (a *ECSService) DescribeInstances(ctx context.Context, request *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
	if err := a.rateLimiter.wait(ctx, ecsDescribeInstances); err != nil {
		return nil, err
	}
	resp, err := a.client.DescribeInstances(request)
	return resp, wrapECSAPIError(ecsDescribeInstances, err)
}

func (a *ECSService) DescribeNetworkInterfaces(ctx context.Context, request *ecs.DescribeNetworkInterfacesRequest) (*ecs.DescribeNetworkInterfacesResponse, error) {
	if err := a.rateLimiter.wait(ctx, ecsDescribeNetworkInterfaces); err != nil {
		return nil, err
	}
	resp, err := a.client.DescribeNetworkInterfaces(request)
	return resp, wrapECSAPIError(ecsDescribeNetworkInterfaces, err)
}

func (a *ECSService) CreateNetworkInterface(ctx context.Context, request *ecs.CreateNetworkInterfaceRequest) (*ecs.CreateNetworkInterfaceResponse, error) {
	if err := a.rateLimiter.wait(ctx, ecsCreateNetworkInterface); err != nil {
		return nil, err
	}
	resp, err := a.client.CreateNetworkInterface(request)
	return resp, wrapECSAPIError(ecsCreateNetworkInterface, err)
}

func (a *ECSService) ModifyNetworkInterfaceAttribute(ctx context.Context, request *ecs.ModifyNetworkInterfaceAttributeRequest) (*ecs.ModifyNetworkInterfaceAttributeResponse, error) {
	if err := a.rateLimiter.wait(ctx, ecsModifyNetworkInterfaceAttribute); err != nil {
		return nil, err
	}
	resp, err := a.client.ModifyNetworkInterfaceAttribute(request)
	return resp, wrapECSAPIError(ecsModifyNetworkInterfaceAttribute, err)
}

func (a *ECSService) TagResources(ctx context.Context, request *ecs.TagResourcesRequest) (*ecs.TagResourcesResponse, error) {
	if err := a.rateLimiter.wait(ctx, ecsTagResources); err != nil {
		return nil, err
	}
	resp, err := a.client.TagResources(request)
	return resp, wrapECSAPIError(ecsTagResources, err)
}

func (a *ECSService) DescribeInstanceTypes(ctx context.Context, request *ecs.DescribeInstanceTypesRequest) (*ecs.DescribeInstanceTypesResponse, error) {
	if err := a.rateLimiter.wait(ctx, ecsDescribeInstanceTypes); err != nil {
		return nil, err
	}
	resp, err := a.client.DescribeInstanceTypes(request)
	return resp, wrapECSAPIError(ecsDescribeInstanceTypes, err)
}

func (a *ECSService) AttachNetworkInterface(ctx context.Context, request *ecs.AttachNetworkInterfaceRequest) (*ecs.AttachNetworkInterfaceResponse, error) {
	if err := a.rateLimiter.wait(ctx, ecsAttachNetworkInterface); err != nil {
		return nil, err
	}
	resp, err := a.client.AttachNetworkInterface(request)
	return resp, wrapECSAPIError(ecsAttachNetworkInterface, err)
}

func (a *ECSService) DescribeInstanceAttribute(ctx context.Context, request *ecs.DescribeInstanceAttributeRequest) (*ecs.DescribeInstanceAttributeResponse, error) {
	if err := a.rateLimiter.wait(ctx, ecsDescribeInstanceAttribute); err != nil {
		return nil, err
	}
	resp, err := a.client.DescribeInstanceAttribute(request)
	return resp, wrapECSAPIError(ecsDescribeInstanceAttribute, err)
}
