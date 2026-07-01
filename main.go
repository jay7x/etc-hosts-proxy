package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	executableName string = "<unknown>"
	logLevel       slog.LevelVar
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
		kv := strings.SplitN(x, "=", 2)
		if len(kv) < 2 {
			continue
		}
		vMap[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return vMap
}

func main() {
	if path, err := os.Executable(); err == nil {
		executableName = filepath.Base(path)
	}

	if err := newApp().Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func newApp() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:               executableName,
		Short:             "Stop changing /etc/hosts file! Use this proxy instead!",
		Version:           version,
		DisableAutoGenTag: true,
	}

	rootCmd.SetVersionTemplate(fmt.Sprintf("%s %s (%s)\n", executableName, version, commit))

	rootCmd.PersistentFlags().String("log-level",
		GetEnvWithDefault("ETC_HOSTS_PROXY_LOG_LEVEL", ""),
		"Set the logging level [debug, info, warn, error]")
	rootCmd.PersistentFlags().Bool("debug",
		GetEnvWithDefault("ETC_HOSTS_PROXY_DEBUG", "false") == "true", "Enable debug mode")
	rootCmd.PersistentFlags().String("log-format",
		GetEnvWithDefault("ETC_HOSTS_PROXY_LOG_FORMAT", ""),
		"Log format [text, json]")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		l, _ := cmd.Flags().GetString("log-level")
		if l != "" {
			var level slog.Level
			switch strings.ToLower(l) {
			case "debug":
				level = slog.LevelDebug
			case "info":
				level = slog.LevelInfo
			case "warn", "warning":
				level = slog.LevelWarn
			case "error":
				level = slog.LevelError
			default:
				return fmt.Errorf("invalid log level %q, expected one of: debug, info, warn, error", l)
			}
			logLevel.Set(level)
		}
		debug, _ := cmd.Flags().GetBool("debug")
		if debug {
			logLevel.Set(slog.LevelDebug)
		}
		format, _ := cmd.Flags().GetString("log-format")
		opts := &slog.HandlerOptions{Level: &logLevel}
		switch strings.ToLower(format) {
		case "json":
			slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, opts)))
		case "text", "":
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))
		default:
			return fmt.Errorf("invalid log format %q, expected one of: text, json", format)
		}

		return nil
	}

	rootCmd.AddCommand(
		newRunCommand(),
	)
	return rootCmd
}
