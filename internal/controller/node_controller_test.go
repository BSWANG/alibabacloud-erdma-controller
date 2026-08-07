package controller

import (
	"context"
	"testing"
	"time"

	networkv1 "github.com/AliyunContainerService/alibabacloud-erdma-controller/api/v1"
	"github.com/AliyunContainerService/alibabacloud-erdma-controller/internal/types"

	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestNodeReconciler_OwnNode(t *testing.T) {
	testcases := []struct {
		name     string
		selector map[string]string
		node     *v1.Node
		expected bool
	}{
		{
			name:     "node is nil",
			selector: nil,
			node:     nil,
			expected: false,
		},
		{
			name:     "selector all nodes",
			selector: nil,
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubernetes.io/nodename": "cn-hangzhou.1.1.1.1",
					},
				},
			},
			expected: true,
		},
		{
			name: "selector match",
			selector: map[string]string{
				"selector1": "value1",
			},
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubernetes.io/nodename": "cn-hangzhou.1.1.1.1",
						"selector1":              "value1",
					},
				},
			},
			expected: true,
		},
		{
			name: "selector partial match",
			selector: map[string]string{
				"selector1": "value1",
				"selector2": "value2",
			},
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubernetes.io/nodename": "cn-hangzhou.1.1.1.1",
						"selector1":              "value1",
					},
				},
			},
			expected: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			r := &NodeReconciler{
				CtrlConfig: &types.Config{
					NodeSelector: tc.selector,
				},
			}
			result := r.OwnNode(tc.node)
			if result != tc.expected {
				t.Errorf("expected %v, but got %v", tc.expected, result)
			}
		})
	}
}

func TestIsNodeReady(t *testing.T) {
	tests := []struct {
		name     string
		node     *v1.Node
		expected bool
	}{
		{
			name: "Node is Ready",
			node: &v1.Node{
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionTrue},
					},
				},
			},
			expected: true,
		},
		{
			name: "Node is NotReady",
			node: &v1.Node{
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionFalse},
					},
				},
			},
			expected: false,
		},
		{
			name:     "No conditions",
			node:     &v1.Node{},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNodeReady(tt.node); got != tt.expected {
				t.Errorf("isNodeReady() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestIsFreshNodeLease(t *testing.T) {
	now := time.Now()
	holder := "node1"
	duration := int32(40)
	freshRenewTime := metav1.NewMicroTime(now.Add(-10 * time.Second))
	staleRenewTime := metav1.NewMicroTime(now.Add(-time.Minute))

	tests := []struct {
		name     string
		lease    *coordinationv1.Lease
		expected bool
	}{
		{
			name: "fresh kubelet lease",
			lease: &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{Name: holder, Namespace: v1.NamespaceNodeLease},
				Spec: coordinationv1.LeaseSpec{
					HolderIdentity:       &holder,
					LeaseDurationSeconds: &duration,
					RenewTime:            &freshRenewTime,
				},
			},
			expected: true,
		},
		{
			name: "expired kubelet lease",
			lease: &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{Name: holder, Namespace: v1.NamespaceNodeLease},
				Spec: coordinationv1.LeaseSpec{
					HolderIdentity:       &holder,
					LeaseDurationSeconds: &duration,
					RenewTime:            &staleRenewTime,
				},
			},
			expected: false,
		},
		{
			name: "non-node lease",
			lease: &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{Name: holder, Namespace: "default"},
				Spec: coordinationv1.LeaseSpec{
					HolderIdentity:       &holder,
					LeaseDurationSeconds: &duration,
					RenewTime:            &freshRenewTime,
				},
			},
			expected: false,
		},
		{
			name: "holder does not match node name",
			lease: &coordinationv1.Lease{
				ObjectMeta: metav1.ObjectMeta{Name: "node2", Namespace: v1.NamespaceNodeLease},
				Spec: coordinationv1.LeaseSpec{
					HolderIdentity:       &holder,
					LeaseDurationSeconds: &duration,
					RenewTime:            &freshRenewTime,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFreshNodeLease(tt.lease, now); got != tt.expected {
				t.Errorf("isFreshNodeLease() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestHasFreshNodeLeaseUsesAPIReader(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}

	now := time.Now()
	holder := "node1"
	duration := int32(40)
	renewTime := metav1.NewMicroTime(now)
	node := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: holder, UID: "node-uid"}}
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      holder,
			Namespace: v1.NamespaceNodeLease,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "v1",
				Kind:       "Node",
				Name:       holder,
				UID:        node.UID,
			}},
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       &holder,
			LeaseDurationSeconds: &duration,
			RenewTime:            &renewTime,
		},
	}
	apiReader := fake.NewClientBuilder().WithScheme(scheme).WithObjects(lease).Build()
	r := &NodeReconciler{
		Client:    fake.NewClientBuilder().WithScheme(scheme).Build(),
		APIReader: apiReader,
	}

	fresh, err := r.hasFreshNodeLease(context.Background(), node, now)
	if err != nil {
		t.Fatalf("hasFreshNodeLease() error = %v", err)
	}
	if !fresh {
		t.Fatal("hasFreshNodeLease() = false, want true")
	}
}

func TestPredictNodeUpdate(t *testing.T) {
	tests := []struct {
		name     string
		oldNode  *v1.Node
		newNode  *v1.Node
		expected bool
	}{
		{
			name: "New node not owned",
			oldNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "test-value",
					},
				},
			},
			newNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "different-value",
					},
				},
			},
			expected: false,
		},
		{
			name: "Old node not owned, new node owned",
			oldNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "not-owned",
					},
				},
			},
			newNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "test-value",
					},
				},
			},
			expected: true,
		},
		{
			name: "Node with deletion timestamp",
			oldNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "test-value",
					},
				},
			},
			newNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "node1",
					DeletionTimestamp: &metav1.Time{},
					Labels: map[string]string{
						"test-key": "test-value",
					},
				},
			},
			expected: true,
		},
		{
			name: "Node becomes Ready",
			oldNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "test-value",
					},
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionFalse},
					},
				},
			},
			newNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "test-value",
					},
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionTrue},
					},
				},
			},
			expected: true,
		},
		{
			name: "Node stays Ready",
			oldNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "test-value",
					},
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionTrue},
					},
				},
			},
			newNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "test-value",
					},
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionTrue},
					},
				},
			},
			expected: false,
		},
		{
			name: "ProviderID changed",
			oldNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "test-value",
					},
				},
				Spec: v1.NodeSpec{
					ProviderID: "old-provider-id",
				},
			},
			newNode: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"test-key": "test-value",
					},
				},
				Spec: v1.NodeSpec{
					ProviderID: "new-provider-id",
				},
			},
			expected: true,
		},
	}

	reconciler := &NodeReconciler{
		CtrlConfig: &types.Config{
			NodeSelector: map[string]string{
				"test-key": "test-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reconciler.PredictNodeUpdate(tt.oldNode, tt.newNode)
			if result != tt.expected {
				t.Errorf("PredictNodeUpdate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestReconcileNodeReadyGate(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = networkv1.AddToScheme(scheme)

	tests := []struct {
		name            string
		node            *v1.Node
		lease           *coordinationv1.Lease
		nodeSelector    map[string]string
		expectRequeue   bool
		expectRequeueAt time.Duration
		expectECS       bool
	}{
		{
			name: "NotReady within timeout requeues",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "node1",
					CreationTimestamp: metav1.Now(),
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionFalse},
					},
				},
			},
			expectRequeue:   true,
			expectRequeueAt: nodeNotReadyRequeueAfter,
		},
		{
			name: "fresh lease does not bypass node selector",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "unselected-node",
					CreationTimestamp: metav1.Now(),
				},
			},
			lease: func() *coordinationv1.Lease {
				holder := "unselected-node"
				duration := int32(40)
				renewTime := metav1.NewMicroTime(time.Now())
				return &coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{Name: holder, Namespace: v1.NamespaceNodeLease},
					Spec: coordinationv1.LeaseSpec{
						HolderIdentity:       &holder,
						LeaseDurationSeconds: &duration,
						RenewTime:            &renewTime,
					},
				}
			}(),
			nodeSelector: map[string]string{"selected": "true"},
		},
		{
			name: "NotReady with fresh lease proceeds",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "node-with-lease",
					UID:               "node-with-lease-uid",
					CreationTimestamp: metav1.Now(),
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionFalse},
					},
				},
			},
			lease: func() *coordinationv1.Lease {
				holder := "node-with-lease"
				duration := int32(40)
				renewTime := metav1.NewMicroTime(time.Now())
				return &coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      holder,
						Namespace: v1.NamespaceNodeLease,
						OwnerReferences: []metav1.OwnerReference{{
							APIVersion: "v1",
							Kind:       "Node",
							Name:       holder,
							UID:        "node-with-lease-uid",
						}},
					},
					Spec: coordinationv1.LeaseSpec{
						HolderIdentity:       &holder,
						LeaseDurationSeconds: &duration,
						RenewTime:            &renewTime,
					},
				}
			}(),
			expectECS: true,
		},
		{
			name: "NotReady with stale lease owner requeues",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "node-with-stale-lease",
					UID:               "current-node-uid",
					CreationTimestamp: metav1.Now(),
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionFalse},
					},
				},
			},
			lease: func() *coordinationv1.Lease {
				holder := "node-with-stale-lease"
				duration := int32(40)
				renewTime := metav1.NewMicroTime(time.Now())
				return &coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      holder,
						Namespace: v1.NamespaceNodeLease,
						OwnerReferences: []metav1.OwnerReference{{
							APIVersion: "v1",
							Kind:       "Node",
							Name:       holder,
							UID:        "previous-node-uid",
						}},
					},
					Spec: coordinationv1.LeaseSpec{
						HolderIdentity:       &holder,
						LeaseDurationSeconds: &duration,
						RenewTime:            &renewTime,
					},
				}
			}(),
			expectRequeue:   true,
			expectRequeueAt: nodeNotReadyRequeueAfter,
		},
		{
			name: "NotReady timeout exceeded proceeds",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "node2",
					CreationTimestamp: metav1.NewTime(time.Now().Add(-10 * time.Minute)),
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionFalse},
					},
				},
			},
			expectECS: true,
		},
		{
			name: "Ready node proceeds",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "node3",
					CreationTimestamp: metav1.Now(),
				},
				Status: v1.NodeStatus{
					Conditions: []v1.NodeCondition{
						{Type: v1.NodeReady, Status: v1.ConditionTrue},
					},
				},
			},
			expectECS: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.node.Spec.ProviderID = "cn-hangzhou.i-test"
			objects := []client.Object{tt.node}
			if tt.lease != nil {
				objects = append(objects, tt.lease)
			}
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				Build()
			api := &fakeEriAPI{}
			r := &NodeReconciler{
				Client:    fakeClient,
				Scheme:    scheme,
				EriClient: &EriClient{client: api, regionID: "cn-hangzhou", ManagedNonOwned: true},
				CtrlConfig: &types.Config{
					WaitNodeReadyTimeoutSeconds: 300,
					NodeSelector:                tt.nodeSelector,
				},
			}

			result, err := r.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: client.ObjectKeyFromObject(tt.node),
			})

			if err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			wantECSCalls := 0
			if tt.expectECS {
				wantECSCalls = 1
			}
			if api.describeInstancesCalls != wantECSCalls {
				t.Fatalf("DescribeInstances calls = %d, want %d", api.describeInstancesCalls, wantECSCalls)
			}
			if tt.expectRequeue {
				if result.RequeueAfter != tt.expectRequeueAt {
					t.Errorf("expected RequeueAfter %v, got %v", tt.expectRequeueAt, result.RequeueAfter)
				}
				return
			}
			if result.RequeueAfter == nodeNotReadyRequeueAfter {
				t.Errorf("should not have requeued with node-ready delay")
			}
		})
	}
}

func TestNodeReconcilerExistingERdmaDeviceSkipsProvisioningAndLeavesCRUnchanged(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}
	if err := networkv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add ERdmaDevice scheme: %v", err)
	}

	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-with-device"},
		Spec:       v1.NodeSpec{ProviderID: "cn-hangzhou.i-current"},
	}
	device := &networkv1.ERdmaDevice{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-with-device",
			Labels: map[string]string{
				"alibabacloud.com/instance-id": "i-existing",
			},
		},
		Spec: networkv1.ERdmaDeviceSpec{
			Devices: []networkv1.DeviceInfo{{ID: "eni-existing"}},
		},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(node, device).
		Build()
	api := &fakeEriAPI{}
	reconciler := &NodeReconciler{
		Client:     fakeClient,
		EriClient:  &EriClient{client: api},
		CtrlConfig: &types.Config{},
	}

	for range 2 {
		if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: client.ObjectKeyFromObject(node),
		}); err != nil {
			t.Fatalf("Reconcile() error = %v", err)
		}
	}
	if api.describeInstancesCalls != 2 {
		t.Fatalf("DescribeInstances calls = %d, want one per reconcile to preserve ERI tag backfill semantics", api.describeInstancesCalls)
	}
	if api.describeInstanceTypesCalls != 0 || api.describeNetworkInterfacesCalls != 0 {
		t.Fatalf("provisioning API calls = instance types %d network interfaces %d, want 0",
			api.describeInstanceTypesCalls, api.describeNetworkInterfacesCalls)
	}
	if api.tagResourcesCalls != 1 {
		t.Fatalf("TagResources calls = %d, want 1 to preserve existing backfill behavior", api.tagResourcesCalls)
	}
	got := &networkv1.ERdmaDevice{}
	if err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(device), got); err != nil {
		t.Fatalf("get ERdmaDevice: %v", err)
	}
	if got.Spec.Devices[0].ID != "eni-existing" ||
		got.Labels["alibabacloud.com/instance-id"] != "i-existing" {
		t.Fatalf("existing ERdmaDevice was changed: %#v", got)
	}
}

func TestNodeReconcilerMissingNodeIgnoresERdmaDeviceNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}
	if err := networkv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add ERdmaDevice scheme: %v", err)
	}

	const nodeName = "missing-node"
	device := &networkv1.ERdmaDevice{
		ObjectMeta: metav1.ObjectMeta{
			Name:       nodeName,
			Labels:     map[string]string{"alibabacloud.com/nodename": nodeName},
			Finalizers: []string{erdmaFinalizer, "example.com/foreign-finalizer"},
		},
	}
	notFound := apierrors.NewNotFound(
		schema.GroupResource{Group: networkv1.GroupVersion.Group, Resource: "erdmadevices"},
		device.Name,
	)
	var patchCalled, deleteCalled bool
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(device).
		WithInterceptorFuncs(interceptor.Funcs{
			Patch: func(_ context.Context, _ client.WithWatch, obj client.Object, _ client.Patch, _ ...client.PatchOption) error {
				patchCalled = true
				if finalizers := obj.GetFinalizers(); len(finalizers) != 1 || finalizers[0] != "example.com/foreign-finalizer" {
					t.Errorf("finalizers after cleanup = %v, want only foreign finalizer", finalizers)
				}
				return notFound
			},
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				deleteCalled = true
				return notFound
			},
		}).
		Build()

	reconciler := &NodeReconciler{Client: fakeClient}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKey{Name: nodeName},
	}); err != nil {
		t.Fatalf("Reconcile() error = %v, want nil", err)
	}
	if !patchCalled || !deleteCalled {
		t.Fatalf("cleanup calls: patch=%t delete=%t, want both true", patchCalled, deleteCalled)
	}
}
