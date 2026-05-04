package main

import (
	"encoding/json"
	"errors"
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
	"kigrepair/internal/winapi"
)

type globalOptions struct {
	output         string
	quiet          bool
	nonInteractive bool
	force          bool
	jsonOutput     bool
	noElevate      bool
	elevatedChild  bool
}

func main() {
	opts := &globalOptions{}
	rootCmd := &cobra.Command{
		Use:           "kigrepair",
		Short:         "Kickidler support and repair utility",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&opts.output, "output", "", "report output directory or report root")
	rootCmd.PersistentFlags().BoolVar(&opts.quiet, "quiet", false, "suppress console output")
	rootCmd.PersistentFlags().BoolVar(&opts.nonInteractive, "non-interactive", false, "disable interactive prompts")
	rootCmd.PersistentFlags().BoolVar(&opts.force, "force", false, "allow future privileged workflows to bypass confirmations")
	rootCmd.PersistentFlags().BoolVar(&opts.jsonOutput, "json", false, "print command results as JSON")
	rootCmd.PersistentFlags().BoolVar(&opts.noElevate, "no-elevate", false, "do not relaunch through UAC when administrator rights are required")
	rootCmd.PersistentFlags().BoolVar(&opts.elevatedChild, "elevated-child", false, "internal flag used after UAC relaunch")
	_ = rootCmd.PersistentFlags().MarkHidden("elevated-child")

	rootCmd.AddCommand(workflowCommand("check", "Detect Kickidler Grabber installation state", opts, checks.CheckWorkflow{}))
	rootCmd.AddCommand(repairCommand(opts))
	rootCmd.AddCommand(cleanupCommand(opts))
	rootCmd.AddCommand(installCommand(opts))
	rootCmd.AddCommand(defenderCommand(opts))
	rootCmd.AddCommand(collectReportCommand(opts))
	rootCmd.AddCommand(versionCommand())

	if err := rootCmd.Execute(); err != nil {
		var exitErr app.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func repairCommand(opts *globalOptions) *cobra.Command {
	var invite string
	var installerPath string
	var yes bool
	cmd := &cobra.Command{
		Use:   "repair",
		Short: "Repair Grabber by cleaning broken state, installing MSI, and ensuring Defender exclusions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflowWithAdmin(cmd, opts, "repair", true, repair.RepairWorkflow{Invite: invite, Installer: installerPath, Yes: yes})
		},
	}
	cmd.Flags().StringVar(&invite, "invite", "", "Kickidler invite string")
	cmd.Flags().StringVar(&installerPath, "installer", "", "path to Grabber installer")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm repair without prompting")
	return cmd
}

func cleanupCommand(opts *globalOptions) *cobra.Command {
	var dryRun bool
	var yes bool
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up Grabber services, processes, files, and registry leftovers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflowWithAdmin(cmd, opts, "cleanup", !dryRun, cleaner.CleanupWorkflow{DryRun: dryRun, Yes: yes})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview cleanup actions without modifying the system")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm real cleanup without prompting")
	return cmd
}

func installCommand(opts *globalOptions) *cobra.Command {
	var invite string
	var installerPath string
	var yes bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Grabber from an MSI package",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflowWithAdmin(cmd, opts, "install", true, installer.InstallWorkflow{Invite: invite, Installer: installerPath, Yes: yes})
		},
	}
	cmd.Flags().StringVar(&invite, "invite", "", "Kickidler invite string")
	cmd.Flags().StringVar(&installerPath, "installer", "", "path to Grabber installer")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm install without prompting")
	return cmd
}

func defenderCommand(opts *globalOptions) *cobra.Command {
	var ensure bool
	var yes bool
	var allKnownPaths bool
	cmd := &cobra.Command{
		Use:   "defender",
		Short: "Inspect or ensure Windows Defender exclusions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflowWithAdmin(cmd, opts, "defender --ensure", ensure, defender.DefenderWorkflow{Ensure: ensure, Yes: yes, AllKnownPaths: allKnownPaths})
		},
	}
	cmd.Flags().BoolVar(&ensure, "ensure", false, "add missing Defender exclusions")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm Defender exclusion changes without prompting")
	cmd.Flags().BoolVar(&allKnownPaths, "all-known-paths", false, "with --ensure, add all configured Defender exclusion paths")
	return cmd
}

func collectReportCommand(opts *globalOptions) *cobra.Command {
	var includeEventLogs bool
	var includeHistory bool
	var historyLimit int
	var noZip bool
	cmd := &cobra.Command{
		Use:   "collect-report",
		Short: "Collect read-only diagnostics and create a support bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflow(opts, diagnostics.CollectReportWorkflow{
				IncludeEventLogs: includeEventLogs,
				IncludeHistory:   includeHistory,
				HistoryLimit:     historyLimit,
				NoZip:            noZip,
			})
		},
	}
	cmd.Flags().BoolVar(&includeEventLogs, "include-eventlogs", true, "include limited Windows Event Log diagnostics")
	cmd.Flags().BoolVar(&includeHistory, "include-history", true, "include recent kigrepair report artifacts")
	cmd.Flags().IntVar(&historyLimit, "history-limit", 5, "number of previous report folders to include")
	cmd.Flags().BoolVar(&noZip, "no-zip", false, "skip creating kigrepair-support-bundle.zip")
	return cmd
}

func workflowCommand(use string, short string, opts *globalOptions, workflow app.Workflow) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflowWithAdmin(cmd, opts, use, false, workflow)
		},
	}
}

type adminCheckerFunc func() bool

func (f adminCheckerFunc) IsAdmin() bool {
	return f()
}

type elevationLauncherFunc func([]string) error

func (f elevationLauncherFunc) RelaunchElevated(args []string) error {
	return f(args)
}

func runWorkflowWithAdmin(cmd *cobra.Command, opts *globalOptions, commandName string, requiresAdmin bool, workflow app.Workflow) error {
	if requiresAdmin && !opts.quiet && !opts.nonInteractive && !opts.noElevate && !opts.elevatedChild && !winapi.IsAdmin() {
		fmt.Fprintln(cmd.ErrOrStderr(), "Administrator rights are required for this command.")
		fmt.Fprintln(cmd.ErrOrStderr(), "Requesting elevation through Windows UAC...")
	}
	result := app.EnsureAdminOrRelaunch(app.AdminGuardOptions{
		CommandName:    commandName,
		RequiresAdmin:  requiresAdmin,
		NoElevate:      opts.noElevate,
		Quiet:          opts.quiet,
		NonInteractive: opts.nonInteractive,
		ElevatedChild:  opts.elevatedChild,
		Args:           os.Args[1:],
		AdminChecker:   adminCheckerFunc(winapi.IsAdmin),
		Launcher:       elevationLauncherFunc(winapi.RelaunchElevated),
	})
	if result.Relaunched {
		return nil
	}
	if result.Err != nil {
		if result.Message != "" && !opts.quiet {
			fmt.Fprintln(cmd.ErrOrStderr(), result.Message)
		}
		return app.ExitError{Code: result.ExitCode}
	}
	return runWorkflow(opts, workflow)
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

	logger, err := logging.New(reports.LogPath(ctx.OutputDir), ctx.Quiet || ctx.JSONOutput)
	if err != nil {
		return err
	}
	defer logger.Close()
	ctx.Logger = logger

	if err := app.RunWorkflow(ctx, workflow); err != nil {
		return err
	}

	if ctx.JSONOutput {
		value := any(ctx.Results)
		if ctx.JSONValue != nil {
			value = ctx.JSONValue
		}
		encoded, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
	} else if !ctx.Quiet {
		fmt.Printf("Report directory: %s\n", ctx.OutputDir)
	}
	if ctx.ExitCode != 0 {
		return app.ExitError{Code: ctx.ExitCode}
	}
	return nil
}
