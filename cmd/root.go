package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/robert-crandall/gh-pr-feedback/internal/app"
	"github.com/spf13/cobra"
)

var (
	flagPR        int
	flagOwner     string
	flagRepo      string
	flagAction    string
	flagOutput    string
	flagPostReply string
	flagCopilot   string
	flagQuiet     bool
)

var rootCmd = &cobra.Command{
	Use:   "gh-pr-feedback",
	Short: "Collect and summarize GitHub PR feedback",
	RunE: func(cmd *cobra.Command, args []string) error {
		action := strings.ToLower(flagAction)
		opts := app.Options{
			PRNumber:   flagPR,
			Owner:      flagOwner,
			Repo:       flagRepo,
			Action:     action,
			OutputPath: flagOutput,
			PostReply:  flagPostReply,
			CopilotCmd: flagCopilot,
			Quiet:      flagQuiet,
			Stdout:     os.Stdout,
			Stderr:     os.Stderr,
		}
		return app.Run(cmd.Context(), opts)
	},
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntVar(&flagPR, "pr", 0, "Override pull request number; auto-detect when omitted")
	rootCmd.Flags().StringVar(&flagOwner, "owner", "", "Repository owner (required when --pr is set without context)")
	rootCmd.Flags().StringVar(&flagRepo, "repo", "", "Repository name (required when --pr is set without context)")
	rootCmd.Flags().StringVar(&flagAction, "action", "markdown", "Action to perform: markdown, raw, summary, or apply")
	rootCmd.Flags().StringVar(&flagOutput, "output", "", "Write Markdown output to the specified file")
	rootCmd.Flags().StringVar(&flagPostReply, "post-reply", "", "Post the provided text as a new PR comment after fetching feedback")
	rootCmd.Flags().StringVar(&flagCopilot, "copilot-cmd", "", "Command used when --action is summary or apply (default: copilot, or copilot --allow-all-tools --allow-all-paths for apply)")
	rootCmd.Flags().BoolVar(&flagQuiet, "quiet", false, "Suppress non-essential log output")
}
