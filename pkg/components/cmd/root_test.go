package cmd

import (
	"testing"

	"github.com/xhanio/framingo/pkg/types/info"

	"github.com/xhanio/gopro/pkg/types"
)

// resetInfo clears the framingo package vars this file asserts on, so one
// case cannot leak into the next.
func resetInfo(t *testing.T) {
	t.Helper()
	old := struct{ name, model, pversion, bversion, tag, module string }{
		info.ProductName, info.ProductModel, info.ProductVersion,
		info.BuildVersion, info.GitTag, info.ProjectName,
	}
	t.Cleanup(func() {
		info.ProductName, info.ProductModel, info.ProductVersion = old.name, old.model, old.pversion
		info.BuildVersion, info.GitTag, info.ProjectName = old.bversion, old.tag, old.module
	})
	info.ProductName, info.ProductModel, info.ProductVersion = "", "", ""
	info.BuildVersion, info.GitTag, info.ProjectName = "", "", ""
}

// project.yaml declares a model, so it has to reach the injected metadata.
// The field was parsed and then never read, leaving info.ProductModel empty
// with no warning for anyone who set it.
func TestApplyProjectInfoCarriesModel(t *testing.T) {
	resetInfo(t)

	applyProjectInfo(types.Project{Product: "demo", Model: "rackmount"})

	if info.ProductModel != "rackmount" {
		t.Errorf("ProductModel = %q, want %q", info.ProductModel, "rackmount")
	}
}

// The --product-model flag still outranks the config, matching how
// --product-version overrides version.
func TestProductModelFlagOverridesConfig(t *testing.T) {
	resetInfo(t)
	oldFlag := productModel
	t.Cleanup(func() { productModel = oldFlag })

	applyProjectInfo(types.Project{Product: "demo", Model: "from-config"})
	productModel = "from-flag"
	overwriteBuildInfo()

	if info.ProductModel != "from-flag" {
		t.Errorf("ProductModel = %q, want the flag to win", info.ProductModel)
	}
}

// An unset flag must leave the configured value alone.
func TestUnsetProductModelFlagKeepsConfig(t *testing.T) {
	resetInfo(t)
	oldFlag := productModel
	t.Cleanup(func() { productModel = oldFlag })

	applyProjectInfo(types.Project{Product: "demo", Model: "from-config"})
	productModel = ""
	overwriteBuildInfo()

	if info.ProductModel != "from-config" {
		t.Errorf("ProductModel = %q, want the config value retained", info.ProductModel)
	}
}

// The surrounding metadata must keep behaving as it did.
func TestApplyProjectInfoExistingFields(t *testing.T) {
	tests := []struct {
		name                                 string
		project                              types.Project
		gitTag                               string
		wantName, wantModule                 string
		wantBuildVersion, wantProductVersion string
	}{
		{
			name:               "explicit version wins",
			project:            types.Project{Product: "demo", Module: "x.test/demo", Version: "v9"},
			gitTag:             "v1",
			wantName:           "demo",
			wantModule:         "x.test/demo",
			wantBuildVersion:   "v1",
			wantProductVersion: "v9",
		},
		{
			name:               "product version falls back to build version",
			project:            types.Project{Product: "demo"},
			gitTag:             "v1",
			wantName:           "demo",
			wantBuildVersion:   "v1",
			wantProductVersion: "v1",
		},
		{
			name:     "no git tag leaves versions empty",
			project:  types.Project{Product: "demo"},
			wantName: "demo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetInfo(t)
			info.GitTag = tt.gitTag

			applyProjectInfo(tt.project)

			if info.ProductName != tt.wantName {
				t.Errorf("ProductName = %q, want %q", info.ProductName, tt.wantName)
			}
			if info.ProjectName != tt.wantModule {
				t.Errorf("ProjectName = %q, want %q", info.ProjectName, tt.wantModule)
			}
			if info.BuildVersion != tt.wantBuildVersion {
				t.Errorf("BuildVersion = %q, want %q", info.BuildVersion, tt.wantBuildVersion)
			}
			if info.ProductVersion != tt.wantProductVersion {
				t.Errorf("ProductVersion = %q, want %q", info.ProductVersion, tt.wantProductVersion)
			}
		})
	}
}
