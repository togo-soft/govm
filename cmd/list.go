package main

import (
	"codeberg.org/application/govm/govm"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

func listCmd() *cobra.Command {
	var stable bool
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "list golang versions",
		Example: "govm list --stable",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd:   true,
			DisableNoDescFlag:   true,
			DisableDescriptions: true,
			HiddenDefaultCmd:    true,
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return mgr.Init()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			versions := mgr.FilterVersions(stable)
			normalizedVersions := make(map[string]string)
			versionList := make([]string, 0, len(versions))

			for _, v := range versions {
				normalized := govm.NormalizeVersion(v)
				normalizedVersions[normalized] = v
				versionList = append(versionList, normalized)
			}

			semver.Sort(versionList)
			for _, normalized := range versionList {
				displayVersion := normalizedVersions[normalized]
				if mgr.Data.IsInstalled(displayVersion) {
					color.Green(displayVersion)
				} else {
					color.White(displayVersion)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&stable, "stable", "s", false, "show stable versions")
	return cmd
}
