package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigConcurrentReconcileSettings(t *testing.T) {
	tests := []struct {
		name          string
		config        string
		expectedNode  int
		expectedERdma int
	}{
		{
			name:          "defaults when omitted",
			config:        `{"region":"cn-test"}`,
			expectedNode:  defaultNodeMaxConcurrentReconciles,
			expectedERdma: defaultERdmaDeviceMaxConcurrentReconciles,
		},
		{
			name:          "uses explicit values",
			config:        `{"region":"cn-test","nodeMaxConcurrentReconciles":3,"erdmaDeviceMaxConcurrentReconciles":7}`,
			expectedNode:  3,
			expectedERdma: 7,
		},
		{
			name:          "replaces non-positive values",
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
		})
	}
}
