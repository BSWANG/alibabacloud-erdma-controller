package controller

import (
	"context"
	"errors"
	"sync"
	"testing"

	networkv1 "github.com/AliyunContainerService/alibabacloud-erdma-controller/api/v1"
	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

type fakeEriAPI struct {
	describeInstancesCalls          int
	describeInstanceTypesCalls      int
	describeNetworkInterfacesCalls  int
	tagResourcesCalls               int
	describeInstanceTypesMu         sync.Mutex
	describeInstanceTypesStarted    chan struct{}
	describeInstanceTypesRelease    chan struct{}
	describeInstanceTypesStartedOne sync.Once
	describeInstanceTypesErr        error
}

func (f *fakeEriAPI) DescribeInstances(context.Context, *ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error) {
	f.describeInstancesCalls++
	return &ecs.DescribeInstancesResponse{
		Body: &ecs.DescribeInstancesResponseBody{
			TotalCount: ptr.To(int32(1)),
			Instances: &ecs.DescribeInstancesResponseBodyInstances{Instance: []*ecs.DescribeInstancesResponseBodyInstancesInstance{{
				InstanceId:   ptr.To("i-test"),
				InstanceType: ptr.To("ecs.g8i.xlarge"),
			}}},
		},
	}, nil
}

func (f *fakeEriAPI) DescribeInstanceTypes(context.Context, *ecs.DescribeInstanceTypesRequest) (*ecs.DescribeInstanceTypesResponse, error) {
	f.describeInstanceTypesMu.Lock()
	f.describeInstanceTypesCalls++
	f.describeInstanceTypesMu.Unlock()
	if f.describeInstanceTypesStarted != nil {
		f.describeInstanceTypesStartedOne.Do(func() { close(f.describeInstanceTypesStarted) })
		<-f.describeInstanceTypesRelease
	}
	if f.describeInstanceTypesErr != nil {
		return nil, f.describeInstanceTypesErr
	}
	return &ecs.DescribeInstanceTypesResponse{
		Body: &ecs.DescribeInstanceTypesResponseBody{
			InstanceTypes: &ecs.DescribeInstanceTypesResponseBodyInstanceTypes{InstanceType: []*ecs.DescribeInstanceTypesResponseBodyInstanceTypesInstanceType{{
				InstanceTypeId:      ptr.To("ecs.g8i.xlarge"),
				EriQuantity:         ptr.To(int32(1)),
				NetworkCardQuantity: ptr.To(int32(1)),
				QueuePairNumber:     ptr.To(int32(2)),
			}}},
		},
	}, nil
}

func (f *fakeEriAPI) DescribeNetworkInterfaces(context.Context, *ecs.DescribeNetworkInterfacesRequest) (*ecs.DescribeNetworkInterfacesResponse, error) {
	f.describeNetworkInterfacesCalls++
	return &ecs.DescribeNetworkInterfacesResponse{
		Body: &ecs.DescribeNetworkInterfacesResponseBody{
			NetworkInterfaceSets: &ecs.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSets{NetworkInterfaceSet: []*ecs.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSet{{
				NetworkInterfaceId:          ptr.To("eni-primary"),
				NetworkInterfaceTrafficMode: ptr.To("HighPerformance"),
				Status:                      ptr.To("InUse"),
				QueuePairNumber:             ptr.To(int32(2)),
				Type:                        ptr.To("Primary"),
				MacAddress:                  ptr.To("00:16:3e:00:00:01"),
				Attachment: &ecs.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSetAttachment{
					NetworkCardIndex: ptr.To(int32(0)),
				},
			}}},
		},
	}, nil
}

func (f *fakeEriAPI) TagResources(context.Context, *ecs.TagResourcesRequest) (*ecs.TagResourcesResponse, error) {
	f.tagResourcesCalls++
	return &ecs.TagResourcesResponse{}, nil
}

func (f *fakeEriAPI) CreateNetworkInterface(context.Context, *ecs.CreateNetworkInterfaceRequest) (*ecs.CreateNetworkInterfaceResponse, error) {
	return nil, errors.New("unexpected CreateNetworkInterface call")
}

func (f *fakeEriAPI) ModifyNetworkInterfaceAttribute(context.Context, *ecs.ModifyNetworkInterfaceAttributeRequest) (*ecs.ModifyNetworkInterfaceAttributeResponse, error) {
	return nil, errors.New("unexpected ModifyNetworkInterfaceAttribute call")
}

func (f *fakeEriAPI) AttachNetworkInterface(context.Context, *ecs.AttachNetworkInterfaceRequest) (*ecs.AttachNetworkInterfaceResponse, error) {
	return nil, errors.New("unexpected AttachNetworkInterface call")
}

func (f *fakeEriAPI) DescribeInstanceAttribute(context.Context, *ecs.DescribeInstanceAttributeRequest) (*ecs.DescribeInstanceAttributeResponse, error) {
	return nil, errors.New("unexpected DescribeInstanceAttribute call")
}

func TestSelectERIsAvoidsRedundantOpenAPICalls(t *testing.T) {
	api := &fakeEriAPI{}
	client := &EriClient{
		client:          api,
		regionID:        "cn-hangzhou",
		ManagedNonOwned: true,
	}
	node := &corev1.Node{Spec: corev1.NodeSpec{ProviderID: "cn-hangzhou.i-test"}}

	instance, err := client.InstanceFromNode(context.Background(), node)
	if err != nil {
		t.Fatalf("InstanceFromNode() error = %v", err)
	}
	for range 2 {
		eris, err := client.SelectERIs(context.Background(), instance)
		if err != nil {
			t.Fatalf("SelectERIs() error = %v", err)
		}
		if len(eris) != 1 || eris[0].ID != "eni-primary" {
			t.Fatalf("SelectERIs() = %#v", eris)
		}
		status, err := client.EnsureEriForInstance(context.Background(), []networkv1.DeviceInfo{{
			ID:           eris[0].ID,
			InstanceID:   eris[0].InstanceID,
			IsPrimaryENI: eris[0].IsPrimaryENI,
			QueuePair:    eris[0].QueuePair,
		}})
		if err != nil {
			t.Fatalf("EnsureEriForInstance() error = %v", err)
		}
		if len(status) != 1 || status[0].Status != networkv1.DeviceStatusReady {
			t.Fatalf("EnsureEriForInstance() = %#v", status)
		}
	}

	if api.describeInstancesCalls != 1 {
		t.Fatalf("DescribeInstances calls = %d, want 1", api.describeInstancesCalls)
	}
	if api.describeInstanceTypesCalls != 1 {
		t.Fatalf("DescribeInstanceTypes calls = %d, want one cached lookup", api.describeInstanceTypesCalls)
	}
	if api.describeNetworkInterfacesCalls != 4 {
		t.Fatalf("DescribeNetworkInterfaces calls = %d, want one per SelectERIs and EnsureEriForInstance call", api.describeNetworkInterfacesCalls)
	}
	if api.tagResourcesCalls != 2 {
		t.Fatalf("TagResources calls = %d, want 2", api.tagResourcesCalls)
	}
}

func TestEriCapacityForInstanceTypeCoalescesConcurrentLookups(t *testing.T) {
	api := &fakeEriAPI{
		describeInstanceTypesStarted: make(chan struct{}),
		describeInstanceTypesRelease: make(chan struct{}),
	}
	client := &EriClient{client: api}
	const callers = 20
	start := make(chan struct{})
	var ready sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(callers)
	done.Add(callers)
	errs := make(chan error, callers)

	for range callers {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			capacity, err := client.eriCapacityForInstanceType(context.Background(), "ecs.g8i.xlarge")
			if err == nil && (!capacity.supported || capacity.cardCount != 1 || capacity.queuePairCount != 2) {
				err = errors.New("unexpected cached capacity")
			}
			errs <- err
		}()
	}
	ready.Wait()
	close(start)
	<-api.describeInstanceTypesStarted
	close(api.describeInstanceTypesRelease)
	done.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("eriCapacityForInstanceType() error = %v", err)
		}
	}
	api.describeInstanceTypesMu.Lock()
	calls := api.describeInstanceTypesCalls
	api.describeInstanceTypesMu.Unlock()
	if calls != 1 {
		t.Fatalf("DescribeInstanceTypes calls = %d, want 1", calls)
	}
}
func TestEriCapacityForInstanceTypeDoesNotCacheErrors(t *testing.T) {
	api := &fakeEriAPI{describeInstanceTypesErr: errors.New("temporary failure")}
	client := &EriClient{client: api}

	if _, err := client.eriCapacityForInstanceType(context.Background(), "ecs.g8i.xlarge"); err == nil {
		t.Fatal("eriCapacityForInstanceType() error = nil, want temporary failure")
	}
	api.describeInstanceTypesErr = nil
	if _, err := client.eriCapacityForInstanceType(context.Background(), "ecs.g8i.xlarge"); err != nil {
		t.Fatalf("eriCapacityForInstanceType() retry error = %v", err)
	}
	if api.describeInstanceTypesCalls != 2 {
		t.Fatalf("DescribeInstanceTypes calls = %d, want retry after error", api.describeInstanceTypesCalls)
	}
}

func TestInstanceFromNodeDoesNotCacheLookup(t *testing.T) {
	api := &fakeEriAPI{}
	client := &EriClient{client: api}
	node := &corev1.Node{Spec: corev1.NodeSpec{ProviderID: "cn-hangzhou.i-test"}}

	for range 2 {
		instance, err := client.InstanceFromNode(context.Background(), node)
		if err != nil {
			t.Fatalf("InstanceFromNode() error = %v", err)
		}
		if instance.InstanceId == nil || *instance.InstanceId != "i-test" {
			t.Fatalf("InstanceFromNode() = %#v", instance)
		}
	}
	if api.describeInstancesCalls != 2 {
		t.Fatalf("DescribeInstances calls = %d, want one per lookup", api.describeInstancesCalls)
	}
}

func TestCreateEriForInstanceWithoutMissingCardsSkipsOpenAPI(t *testing.T) {
	client := &EriClient{}
	eris, err := client.CreateEriForInstance(context.Background(), nil, nil, 0)
	if err != nil {
		t.Fatalf("CreateEriForInstance() error = %v", err)
	}
	if len(eris) != 0 {
		t.Fatalf("CreateEriForInstance() = %#v, want no ERIs", eris)
	}
}
