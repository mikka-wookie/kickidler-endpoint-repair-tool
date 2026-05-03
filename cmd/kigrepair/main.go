package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"kigrepair/internal/app"
	"kigrepair/internal/checks"
	"kigrepair/internal/cleaner"
	"kigrepair/internal/config"
	"kigrepair/internal/defender"
	"kigrepair/internal/diagnostics"
	"kigrepair/internal/installer"
	"kigrepair/internal/logging"
	"kigrepair/internal/repair"
	"kigrepair/internal/reports"
)

type globalOptions struct {
	output         string
	quiet          bool
	nonInteractive bool
	force          bool
	jsonOutput     bool
}

func main() {
	opts := &globalOptions{}
	rootCmd := &cobra.Command{
		Use:   "kigrepair",
		Short: "Kickidler support and repair utility",
	}

	rootCmd.PersistentFlags().StringVar(&opts.output, "output", "", "report output directory or report root")
	rootCmd.PersistentFlags().BoolVar(&opts.quiet, "quiet", false, "suppress console output")
	rootCmd.PersistentFlags().BoolVar(&opts.nonInteractive, "non-interactive", false, "disable interactive prompts")
	rootCmd.PersistentFlags().BoolVar(&opts.force, "force", false, "allow future privileged workflows to bypass confirmations")
	rootCmd.PersistentFlags().BoolVar(&opts.jsonOutput, "json", false, "print command results as JSON")

	rootCmd.AddCommand(workflowCommand("check", "Run placeholder checks", opts, checks.CheckWorkflow{}))
	rootCmd.AddCommand(repairCommand(opts))
	rootCmd.AddCommand(workflowCommand("cleanup", "Plan placeholder cleanup", opts, cleaner.CleanupWorkflow{}))
	rootCmd.AddCommand(installCommand(opts))
	rootCmd.AddCommand(defenderCommand(opts))
	rootCmd.AddCommand(workflowCommand("collect-report", "Collect placeholder diagnostics report", opts, diagnostics.CollectReportWorkflow{}))
	rootCmd.AddCommand(versionCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func repairCommand(opts *globalOptions) *cobra.Command {
	var invite string
	var installerPath string
	cmd := &cobra.Command{
		Use:   "repair",
		Short: "Run placeholder Grabber repair workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflow(opts, repair.RepairWorkflow{Invite: invite, Installer: installerPath})
		},
	}
	cmd.Flags().StringVar(&invite, "invite", "", "Kickidler invite string")
	cmd.Flags().StringVar(&installerPath, "installer", "", "path to Grabber installer")
	return cmd
}

func installCommand(opts *globalOptions) *cobra.Command {
	var invite string
	var installerPath string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Run placeholder Grabber install workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflow(opts, installer.InstallWorkflow{Invite: invite, Installer: installerPath})
		},
	}
	cmd.Flags().StringVar(&invite, "invite", "", "Kickidler invite string")
	cmd.Flags().StringVar(&installerPath, "installer", "", "path to Grabber installer")
	return cmd
}

func defenderCommand(opts *globalOptions) *cobra.Command {
	var ensure bool
	cmd := &cobra.Command{
		Use:   "defender",
		Short: "Inspect placeholder Defender configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflow(opts, defender.DefenderWorkflow{Ensure: ensure})
		},
	}
	cmd.Flags().BoolVar(&ensure, "ensure", false, "placeholder flag for future Defender exclusion enforcement")
	return cmd
}

func workflowCommand(use string, short string, opts *globalOptions, workflow app.Workflow) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflow(opts, workflow)
		},
	}
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s %s\n", config.BinaryName, config.Version)
		},
	}
}

func runWorkflow(opts *globalOptions, workflow app.Workflow) error {
	ctx := app.NewContext()
	ctx.StartedAt = time.Now()
	ctx.ReportRoot = config.DefaultReportRoot
	ctx.Quiet = opts.quiet
	ctx.NonInteractive = opts.nonInteractive
	ctx.Force = opts.force
	ctx.JSONOutput = opts.jsonOutput
	if opts.quiet {
		ctx.Mode = app.RunModeQuiet
	}
	if opts.output != "" {
		ctx.OutputDir = opts.output
	} else {
		ctx.OutputDir = reports.TimestampedDir(ctx.ReportRoot, ctx.StartedAt)
	}

	reporter, err := reports.New(ctx.OutputDir)
	if err != nil {
		return err
	}
	ctx.Reporter = reporter

	logger, err := logging.New(reports.LogPath(ctx.OutputDir), ctx.Quiet)
	if err != nil {
		return err
	}
	defer logger.Close()
	ctx.Logger = logger

	if err := app.RunWorkflow(ctx, workflow); err != nil {
		return err
	}

	if ctx.JSONOutput {
		encoded, err := json.MarshalIndent(ctx.Results, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
	} else if !ctx.Quiet {
		fmt.Printf("Report directory: %s\n", ctx.OutputDir)
	}
	return nil
}
