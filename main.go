package main

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	executableName string = "<unknown>"
)

func main() {
	if path, err := os.Executable(); err == nil {
		executableName = filepath.Base(path)
	}

	if err := newApp().Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func newApp() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               executableName,
		Short:             "Stop changing /etc/hosts file! Use this proxy instead!",
		Version:           version,
		DisableAutoGenTag: true,
	}

	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.PersistentFlags().String("log-level", "", "Set the logging level [trace, debug, info, warn, error]")
	rootCmd.PersistentFlags().Bool("debug", false, "debug mode")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		l, _ := cmd.Flags().GetString("log-level")
		if l != "" {
			lvl, err := logrus.ParseLevel(l)
			if err != nil {
				return err
			}
			logrus.SetLevel(lvl)
		}
		debug, _ := cmd.Flags().GetBool("debug")
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		return nil
	}

	rootCmd.AddCommand(
		newRunCommand(),
	)
	return rootCmd
}
