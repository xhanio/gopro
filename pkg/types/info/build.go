package info

import (
	"fmt"
	"strings"

	"github.com/xhanio/framingo/pkg/utils/sliceutil"
)

var (
	ProductName    string
	ProductModel   string
	ProductVersion string

	ProjectRoot string
	ProjectName string
	ProjectPath string

	GitBranch string
	GitTag    string

	BuildVersion string
	BuildType    string
	BuildDate    string
	BuildTime    string

	INJECTION = map[string]*string{
		"ProductName":    &ProductName,
		"ProductModel":   &ProductModel,
		"ProductVersion": &ProductVersion,

		"ProjectRoot": &ProjectRoot,
		"ProjectName": &ProjectName,
		"ProjectPath": &ProjectPath,

		"GitBranch": &GitBranch,
		"GitTag":    &GitTag,

		"BuildVersion": &BuildVersion,
		"BuildType":    &BuildType,
		"BuildDate":    &BuildDate,
		"BuildTime":    &BuildTime,
	}
)

func Version() string {
	var version []string
	version = append(version, ProductVersion, BuildVersion)
	version = sliceutil.Deduplicate(version...)
	if BuildType != "" {
		version = append(version, fmt.Sprintf("(%s)", BuildType))
	}
	return strings.Join(version, " ")
}
