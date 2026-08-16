package cmd

import (
	"testing"

	"github.com/xhanio/framingo/pkg/types/info"

	"github.com/xhanio/gopro/pkg/types"
)

// Each build.binaries entry is one application, so the binary being built has
// to reach the injected application identity.
func TestApplyApplicationInfo(t *testing.T) {
	tests := []struct {
		name           string
		binary         types.BinarySpec
		productVersion string
		wantName       string
		wantVersion    string
	}{
		{
			name:           "binary version wins",
			binary:         types.BinarySpec{Name: "api", Version: "v2.1.0"},
			productVersion: "v9",
			wantName:       "api",
			wantVersion:    "v2.1.0",
		},
		{
			name:           "unset version falls back to product version",
			binary:         types.BinarySpec{Name: "worker"},
			productVersion: "v9",
			wantName:       "worker",
			wantVersion:    "v9",
		},
		{
			name:        "no versions anywhere stays empty",
			binary:      types.BinarySpec{Name: "cli"},
			wantName:    "cli",
			wantVersion: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetInfo(t)
			info.ProductVersion = tt.productVersion

			applyApplicationInfo(tt.binary)

			if info.ApplicationName != tt.wantName {
				t.Errorf("ApplicationName = %q, want %q", info.ApplicationName, tt.wantName)
			}
			if info.ApplicationVersion != tt.wantVersion {
				t.Errorf("ApplicationVersion = %q, want %q", info.ApplicationVersion, tt.wantVersion)
			}
		})
	}
}
