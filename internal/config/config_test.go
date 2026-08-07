package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigControllerSettings(t *testing.T) {
	tests := []struct {
		name                 string
		config               string
		expectedNode         int
		expectedERdma        int
		expectedECSPerMinute int
	}{
		{
			name:          "defaults when omitted",
			config:        `{"region":"cn-test"}`,
			expectedNode:  defaultNodeMaxConcurrentReconciles,
			expectedERdma: defaultERdmaDeviceMaxConcurrentReconciles,
		},
		{
			name:                 "uses simple per-minute override",
			config:               `{"region":"cn-test","nodeMaxConcurrentReconciles":3,"erdmaDeviceMaxConcurrentReconciles":7,"rateLimit":{"CreateNetworkInterface":630}}`,
			expectedNode:         3,
			expectedERdma:        7,
			expectedECSPerMinute: 630,
		},
		{
			name:          "replaces non-positive concurrency values",
			config:        `{"region":"cn-test","nodeMaxConcurrentReconciles":-1,"erdmaDeviceMaxConcurrentReconciles":0}`,
			expectedNode:  defaultNodeMaxConcurrentReconciles,
			expectedERdma: defaultERdmaDeviceMaxConcurrentReconciles,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tt.config), 0o600); err != nil {
				t.Fatalf("write config: %v", err)
			}
			got, err := parseConfig(path)
			if err != nil {
				t.Fatalf("parseConfig() error = %v", err)
			}
			if got.NodeMaxConcurrentReconciles != tt.expectedNode {
				t.Errorf("NodeMaxConcurrentReconciles = %d, want %d", got.NodeMaxConcurrentReconciles, tt.expectedNode)
			}
			if got.ERdmaDeviceMaxConcurrentReconciles != tt.expectedERdma {
				t.Errorf("ERdmaDeviceMaxConcurrentReconciles = %d, want %d", got.ERdmaDeviceMaxConcurrentReconciles, tt.expectedERdma)
			}
			if tt.expectedECSPerMinute > 0 && got.RateLimit["CreateNetworkInterface"] != tt.expectedECSPerMinute {
				t.Errorf("CreateNetworkInterface requests/minute = %v, want %v", got.RateLimit["CreateNetworkInterface"], tt.expectedECSPerMinute)
			}
		})
	}
}

func TestParseConfigBurstSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{"region":"cn-test","rateLimitBurst":{"DescribeInstances":1}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	got, err := parseConfig(path)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}
	if got.RateLimitBurst["DescribeInstances"] != 1 {
		t.Fatalf("DescribeInstances burst = %d, want 1", got.RateLimitBurst["DescribeInstances"])
	}
}
