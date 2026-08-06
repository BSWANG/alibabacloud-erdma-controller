package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigControllerSettings(t *testing.T) {
	tests := []struct {
		name               string
		config             string
		expectedNode       int
		expectedERdma      int
		expectedLimitCount int
		expectedPerMinute  int
	}{
		{
			name:          "defaults when omitted",
			config:        `{"region":"cn-test"}`,
			expectedNode:  defaultNodeMaxConcurrentReconciles,
			expectedERdma: defaultERdmaDeviceMaxConcurrentReconciles,
		},
		{
			name:               "uses per API limit",
			config:             `{"region":"cn-test","nodeMaxConcurrentReconciles":3,"erdmaDeviceMaxConcurrentReconciles":7,"rateLimit":{"CreateNetworkInterface":600}}`,
			expectedNode:       3,
			expectedERdma:      7,
			expectedLimitCount: 1,
			expectedPerMinute:  600,
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
			if len(got.RateLimit) != tt.expectedLimitCount {
				t.Fatalf("len(RateLimit) = %d, want %d", len(got.RateLimit), tt.expectedLimitCount)
			}
			if tt.expectedLimitCount > 0 && got.RateLimit["CreateNetworkInterface"] != tt.expectedPerMinute {
				t.Errorf("CreateNetworkInterface limit = %d, want %d", got.RateLimit["CreateNetworkInterface"], tt.expectedPerMinute)
			}
		})
	}
}
