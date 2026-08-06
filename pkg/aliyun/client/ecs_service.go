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

const (
	APIDescribeInstances               = "DescribeInstances"
	APIDescribeNetworkInterfaces       = "DescribeNetworkInterfaces"
	APICreateNetworkInterface          = "CreateNetworkInterface"
	APIModifyNetworkInterfaceAttribute = "ModifyNetworkInterfaceAttribute"
	APITagResources                    = "TagResources"
	APIDescribeInstanceTypes           = "DescribeInstanceTypes"
	APIAttachNetworkInterface          = "AttachNetworkInterface"
	APIDescribeInstanceAttribute       = "DescribeInstanceAttribute"
)

// ECSAPI is the subset of the ECS SDK used by ECSService.
type ECSAPI interface {
	DescribeInstances(*ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error)
	DescribeNetworkInterfaces(*ecs.DescribeNetworkInterfacesRequest) (*ecs.DescribeNetworkInterfacesResponse, error)
	CreateNetworkInterface(*ecs.CreateNetworkInterfaceRequest) (*ecs.CreateNetworkInterfaceResponse, error)
	ModifyNetworkInterfaceAttribute(*ecs.ModifyNetworkInterfaceAttributeRequest) (*ecs.ModifyNetworkInterfaceAttributeResponse, error)
	TagResources(*ecs.TagResourcesRequest) (*ecs.TagResourcesResponse, error)
	DescribeInstanceTypes(*ecs.DescribeInstanceTypesRequest) (*ecs.DescribeInstanceTypesResponse, error)
	AttachNetworkInterface(*ecs.AttachNetworkInterfaceRequest) (*ecs.AttachNetworkInterfaceResponse, error)
	DescribeInstanceAttribute(*ecs.DescribeInstanceAttributeRequest) (*ecs.DescribeInstanceAttributeResponse, error)
}

// ECSService follows Terway's ECSService pattern: every SDK method waits on
// the shared per-API RateLimiter before issuing the request.
type ECSService struct {
	client      ECSAPI
	rateLimiter *RateLimiter
}

func NewECSService(client ECSAPI, rateLimiter *RateLimiter) *ECSService {
	return &ECSService{
		client:      client,
		rateLimiter: rateLimiter,
	}
}

func (a *ECSService) DescribeInstances(ctx context.Context, request *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
	if err := a.rateLimiter.Wait(ctx, APIDescribeInstances); err != nil {
		return nil, err
	}
	return a.client.DescribeInstances(request)
}

func (a *ECSService) DescribeNetworkInterfaces(ctx context.Context, request *ecs.DescribeNetworkInterfacesRequest) (*ecs.DescribeNetworkInterfacesResponse, error) {
	if err := a.rateLimiter.Wait(ctx, APIDescribeNetworkInterfaces); err != nil {
		return nil, err
	}
	return a.client.DescribeNetworkInterfaces(request)
}

func (a *ECSService) CreateNetworkInterface(ctx context.Context, request *ecs.CreateNetworkInterfaceRequest) (*ecs.CreateNetworkInterfaceResponse, error) {
	if err := a.rateLimiter.Wait(ctx, APICreateNetworkInterface); err != nil {
		return nil, err
	}
	return a.client.CreateNetworkInterface(request)
}

func (a *ECSService) ModifyNetworkInterfaceAttribute(ctx context.Context, request *ecs.ModifyNetworkInterfaceAttributeRequest) (*ecs.ModifyNetworkInterfaceAttributeResponse, error) {
	if err := a.rateLimiter.Wait(ctx, APIModifyNetworkInterfaceAttribute); err != nil {
		return nil, err
	}
	return a.client.ModifyNetworkInterfaceAttribute(request)
}

func (a *ECSService) TagResources(ctx context.Context, request *ecs.TagResourcesRequest) (*ecs.TagResourcesResponse, error) {
	if err := a.rateLimiter.Wait(ctx, APITagResources); err != nil {
		return nil, err
	}
	return a.client.TagResources(request)
}

func (a *ECSService) DescribeInstanceTypes(ctx context.Context, request *ecs.DescribeInstanceTypesRequest) (*ecs.DescribeInstanceTypesResponse, error) {
	if err := a.rateLimiter.Wait(ctx, APIDescribeInstanceTypes); err != nil {
		return nil, err
	}
	return a.client.DescribeInstanceTypes(request)
}

func (a *ECSService) AttachNetworkInterface(ctx context.Context, request *ecs.AttachNetworkInterfaceRequest) (*ecs.AttachNetworkInterfaceResponse, error) {
	if err := a.rateLimiter.Wait(ctx, APIAttachNetworkInterface); err != nil {
		return nil, err
	}
	return a.client.AttachNetworkInterface(request)
}

func (a *ECSService) DescribeInstanceAttribute(ctx context.Context, request *ecs.DescribeInstanceAttributeRequest) (*ecs.DescribeInstanceAttributeResponse, error) {
	if err := a.rateLimiter.Wait(ctx, APIDescribeInstanceAttribute); err != nil {
		return nil, err
	}
	return a.client.DescribeInstanceAttribute(request)
}
