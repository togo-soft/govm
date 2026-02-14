package main

import (
	"log/slog"
	"os"

	"codefloe.com/apps/govm/govm"
	"codefloe.com/apps/govm/internal/fsutil"
	"codefloe.com/apps/govm/logger"
	"github.com/spf13/cobra"
)

var mgr = govm.NewManager()

func main() {
	cleanup := logger.Setup(fsutil.WorkspaceDir(), "govm.log")
	defer cleanup()

	root := &cobra.Command{
		Use:   "govm",
		Short: "go version management",
		CompletionOptions: cobra.CompletionOptions{
			DisableNoDescFlag:   true,
			DisableDescriptions: true,
			HiddenDefaultCmd:    true,
		},
		Run: func(cmd *cobra.Command, args []string) {},
	}

	root.AddCommand(listCmd(), useCmd(), removeCmd())

	if err := root.Execute(); err != nil {
		slog.Error("execute command failed", "reason", err)
		os.Exit(1)
	}
}
