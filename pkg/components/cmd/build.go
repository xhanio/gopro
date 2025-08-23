package cmd

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/xhanio/framingo/pkg/types/info"
)

var (
	productModel   string
	productVersion string
	buildVersion   string
	buildType      string
	buildDate      string
	binaryOutput   string

	pushImage bool
)

func NewBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "build",
	}
	cmd.AddCommand(NewBuildBinaryCmd())
	cmd.AddCommand(NewBuildImageCmd())
	return cmd
}

func NewBuildBinaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "binary",
		RunE: runBuildBinary,
	}
	cmd.Flags().StringVarP(&productModel, "product-model", "", "", "overwrite product model")
	cmd.Flags().StringVarP(&productVersion, "product-version", "", "", "overwrite product version")
	cmd.Flags().StringVarP(&buildVersion, "build-version", "", "", "overwrite build version")
	cmd.Flags().StringVarP(&buildType, "build-type", "", "", "overwrite build type")
	cmd.Flags().StringVarP(&buildDate, "build-date", "", "", "overwrite build date")
	cmd.Flags().StringVarP(&binaryOutput, "output", "o", "", "build binary output dir")
	return cmd
}

func overwriteBuildInfo() {
	if productModel != "" {
		info.ProductModel = productModel
	}
	if productVersion != "" {
		info.ProductVersion = productVersion
	}
	if buildVersion != "" {
		info.BuildVersion = buildVersion
	}
	if buildType != "" {
		info.BuildType = buildType
	}
	if buildDate != "" {
		info.BuildDate = buildDate
	}
}

func runBuildBinary(cmd *cobra.Command, args []string) error {
	overwriteBuildInfo()
	if binaryOutput == "" {
		binaryOutput = envConf.BinaryTgt
	}
	for _, name := range envConf.Binaries {
		if !filterRegex.MatchString(name) {
			continue
		}
		for _, binary := range conf.Build.Binaries {
			if name != binary.Name {
				continue
			}
			binarySrc := binary.Src
			if binarySrc == "" {
				binarySrc = filepath.Join(envConf.BinarySrc, binary.Name)
			}
			// build default platform
			titlef("Build Binary %s from %s", name, binarySrc)
			if err := executeBuildBinary(binary.Name, "", binarySrc, binaryOutput); err != nil {
				return err
			}
			for _, platform := range binary.Platform {
				linef("build for platform %s", platform)
				if err := executeBuildBinary(binary.Name, platform, binarySrc, binaryOutput); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func NewBuildImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "image",
		RunE: runBuildImage,
	}
	cmd.Flags().BoolVarP(&pushImage, "push", "p", false, "push image")
	return cmd
}

func runBuildImage(cmd *cobra.Command, args []string) error {
	for _, name := range envConf.Images {
		if !filterRegex.MatchString(name) {
			continue
		}
		for _, image := range conf.Build.Images {
			if name != image.Name {
				continue
			}
			buildTarget := image.GetImageName(envConf)
			if image.BuildFrom != "" {
				// build from thrid party image
				buildSource := image.BuildFrom
				titlef("Build Image %s from %s as %s", name, buildSource, buildTarget)
				err := executePullImage(buildSource)
				if err != nil {
					return err
				}
				err = executeTagImage(buildSource, buildTarget)
				if err != nil {
					return err
				}
			} else {
				// build from dockerfile
				buildSource := image.BuildSrc
				if buildSource == "" {
					buildSource = filepath.Join(envConf.ImageBuildSrc, image.Name)
				}
				titlef("Build Image %s from %s as %s", name, buildSource, buildTarget)
				buildBase := image.Base
				if baseName, ok := strings.CutPrefix(buildBase, "$"); ok {
					buildBase = GetImageName(baseName)
				}
				err := executeBuildImage(name, buildSource, buildTarget, buildBase)
				if err != nil {
					return err
				}
			}
			if pushImage && !image.NoPush {
				titlef("Push Image %s", buildTarget)
				err := executePushImage(buildTarget)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
