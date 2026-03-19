package cmd

import (
	"fmt"
	"os"

	"github.com/dumbmachine/db-cli/cmd/annotate"
	"github.com/dumbmachine/db-cli/cmd/connection"
	"github.com/dumbmachine/db-cli/cmd/query"
	"github.com/dumbmachine/db-cli/cmd/schema"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	outputFormat string
	fields       string
	limit        int
	offset       int
	timeout      string
	yes          bool
	noColor      bool
	quiet        bool
	verbose      bool
	connFlag     string
)

var rootCmd = &cobra.Command{
	Use:   "dq",
	Short: "Agent-first database CLI",
	Long:  `dq is an agent-friendly database tool for discovering, querying, introspecting, and annotating databases.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !cmd.Flags().Changed("output") && outputFormat == "" {
			outputFormat = output.DefaultFormat()
		}
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		output.PrintError("error", err.Error(), "")
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, table, csv, ndjson (default: table for TTY, json for pipes)")
	rootCmd.PersistentFlags().StringVar(&fields, "fields", "", "Comma-separated list of fields to include")
	rootCmd.PersistentFlags().IntVar(&limit, "limit", 0, "Maximum number of rows to return")
	rootCmd.PersistentFlags().IntVar(&offset, "offset", 0, "Number of rows to skip")
	rootCmd.PersistentFlags().StringVar(&timeout, "timeout", "30s", "Query timeout duration")
	rootCmd.PersistentFlags().BoolVar(&yes, "yes", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVarP(&connFlag, "connection", "c", "", "Connection name to use")

	// Register subcommands
	rootCmd.AddCommand(connection.Cmd)
	rootCmd.AddCommand(schema.Cmd)
	rootCmd.AddCommand(annotate.Cmd)
	query.RegisterCommands(rootCmd)
}

func GetOutputFormat() string {
	if outputFormat == "" {
		return output.DefaultFormat()
	}
	return outputFormat
}

func GetConnection(cmd *cobra.Command) string {
	if connFlag == "" {
		fmt.Fprintln(os.Stderr, `{"error":"usage","message":"--connection (-c) flag is required","suggestion":"Use -c <name> to specify a connection"}`)
		os.Exit(2)
	}
	return connFlag
}
