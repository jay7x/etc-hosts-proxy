package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	executableName string = "<unknown>"
)

func GetEnvWithDefault(name string, defaultValue string) string {
	ev, found := os.LookupEnv(name)
	if !found {
		return defaultValue
	}
	return ev
}

func GetEnvStrMap(name string) map[string]string {
	ev, found := os.LookupEnv(name)
	if !found {
		return map[string]string{}
	}
	vList := strings.Split(ev, ",")
	vMap := make(map[string]string, len(vList))
	for _, x := range vList {
		kv := strings.Split(x, "=")
		vMap[kv[0]] = kv[1]
	}
	return vMap
}

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

	rootCmd.PersistentFlags().String("log-level",
		GetEnvWithDefault("ETC_HOSTS_PROXY_LOG_LEVEL", ""),
		"Set the logging level [trace, debug, info, warn, error]")
	rootCmd.PersistentFlags().Bool("debug",
		GetEnvWithDefault("ETC_HOSTS_PROXY_DEBUG", "false") == "true", "Enable debug mode")

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
