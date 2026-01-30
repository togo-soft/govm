package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func (m *Command) useCommand() {
	var version, site string
	var vm = NewVersionManager()
	cmd := &cobra.Command{
		Use:     "use [version]",
		Short:   "switch or install golang version",
		Example: "govm use 1.24.11\ngovm use 1.24.11 -s https://golang.google.cn/dl/\ngovm use -v 1.24.11 -s https://golang.google.cn/dl/\ncommon download sites:\n\thttps://go.dev/dl/\n\thttps://golang.google.cn/dl/\n\thttps://mirrors.aliyun.com/golang/\n\thttps://mirrors.hust.edu.cn/golang/\n\thttps://mirrors.nju.edu.cn/golang/",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return vm.Initialized()
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// If version not specified via flag, try to get it from positional argument
			if version == "" && len(args) > 0 {
				version = args[0]
			}
			if vm.IsValidVersion(version) {
				return nil
			}
			return errors.New("invalid version: version not found")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return vm.InstallVersion(version, site)
		},
	}

	cmd.Flags().StringVarP(&version, "version", "v", "", "switch or install golang version")
	cmd.Flags().StringVarP(&site, "site", "s", "https://go.dev/dl", "download go version site")

	m.cmd.AddCommand(cmd)
}
