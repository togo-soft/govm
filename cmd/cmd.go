package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"gitter.top/apps/govm/logger"
)

var log *logger.Logger

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("getting user home directory failed", "reason", err)
		homeDir = "."
	}
	logName := "govm.log"
	log = logger.NewLogger(homeDir, logName)
}

type Command struct {
	cmd *cobra.Command
}

func (m *Command) rootCommand() {
	m.cmd = &cobra.Command{
		Use:   "govm",
		Short: "go version management",
		CompletionOptions: cobra.CompletionOptions{
			DisableNoDescFlag:   true,
			DisableDescriptions: true,
			HiddenDefaultCmd:    true,
		},
		Run: func(cmd *cobra.Command, args []string) {},
	}
}

func (m *Command) load() {
	m.rootCommand()
	m.listCommand()
}

func (m *Command) Execute() error {
	return m.cmd.Execute()
}

func main() {
	var command = &Command{}
	command.load()
	if err := command.Execute(); err != nil {
		log.Error("execute command failed", "reason", err)
	}
}
