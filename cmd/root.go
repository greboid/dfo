package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/greboid/dfo/pkg/packages"
	"github.com/spf13/cobra"
)

var (
	alpineClient = packages.NewAlpineClient()
	debugMode    bool
)

var rootCmd = &cobra.Command{
	Use:   "dfo",
	Short: "Generate contempt templates from YAML build files",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := slog.LevelInfo
		if debugMode {
			level = slog.LevelDebug
		}
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		}))
		slog.SetDefault(logger)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
