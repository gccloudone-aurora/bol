package cmd

import (
	"context"
	"log"

	"github.com/gccloudone-aurora/bol/pkg/report"
	"github.com/gccloudone-aurora/bol/pkg/util"
	"github.com/spf13/cobra"
)

var configPath string

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate the finance extract",
	Long:  `Generate a daily finance extract for the given days.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		config, err := util.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		rprt, err := report.NewReport(ctx, *config)
		rprt.Generate(ctx)

	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVar(&configPath, "config-path", "", "The config file path")
}
