// cmd/root.go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jackchuka/gv/internal/config"
	"github.com/jackchuka/gv/tui"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "gv",
	Short: "Git Vision - monitor multiple git repositories",
	Long: `
            ╔═╗╦  ╦
            ║ ╦╚╗╔╝
            ╚═╝ ╚╝   Git Vision

  TUI dashboard for monitoring multiple git repositories
  and worktrees. Auto-discovers git repos under configured
  paths and shows their status in real-time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(cfg.ScanPaths) == 0 {
			fmt.Fprintf(os.Stderr, "No scan paths configured.\n")
			fmt.Fprintf(os.Stderr, "Run 'gv init' to set up, or add paths to %s\n", cfgFile)
			return nil
		}

		return tui.Run(cfg)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/gv/config.yaml)")
	rootCmd.PersistentFlags().StringSliceP("scan", "s", nil, "paths to scan (overrides config file)")
}

func initConfig() {
	if cfgFile == "" {
		cfgFile = config.DefaultConfigPath()
	}

	var err error
	cfg, err = config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Override scan paths if provided via flags
	if scanPaths, _ := rootCmd.Flags().GetStringSlice("scan"); len(scanPaths) > 0 {
		cfg.ScanPaths = scanPaths
	}
}
