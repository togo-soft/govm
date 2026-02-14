package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func removeCmd() *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:     "remove [version]",
		Short:   "remove golang version",
		Example: "govm remove 1.24.11\ngovm remove -v 1.24.11",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return mgr.Init()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if version == "" && len(args) > 0 {
				version = args[0]
			}
			if version == "" {
				return errors.New("version is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return mgr.Uninstall(version)
		},
	}

	cmd.Flags().StringVarP(&version, "version", "v", "", "golang version to remove")
	return cmd
}
