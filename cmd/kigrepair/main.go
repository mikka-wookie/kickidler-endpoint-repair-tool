package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
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
	"kigrepair/internal/preflight"
	"kigrepair/internal/repair"
	"kigrepair/internal/reports"
	"kigrepair/internal/verifier"
	"kigrepair/internal/version"
	"kigrepair/internal/winapi"
	"kigrepair/internal/wizard"
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
	rootCmd.AddCommand(preflightCommand(opts))
	rootCmd.AddCommand(cleanupCommand(opts))
	rootCmd.AddCommand(installCommand(opts))
	rootCmd.AddCommand(defenderCommand(opts))
	rootCmd.AddCommand(collectReportCommand(opts))
	rootCmd.AddCommand(reportsCommand(opts))
	rootCmd.AddCommand(verifyCommand(opts))
	rootCmd.AddCommand(wizardCommand(opts))
	rootCmd.AddCommand(versionCommand(opts))

	if err := rootCmd.Execute(); err != nil {
		var exitErr app.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(app.ExitUnexpectedError)
	}
}

func wizardCommand(opts *globalOptions) *cobra.Command {
	var invite string
	var installerPath string
	var collectBundle bool
	var allowRepair bool
	var verifyOnly bool
	var yes bool
	cmd := &cobra.Command{
		Use:     "wizard",
		Aliases: []string{"support"},
		Short:   "Run an interactive support wizard",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflow(opts, wizard.Workflow{Options: wizard.Options{
				InstallerPath:  installerPath,
				HasInvite:      strings.TrimSpace(invite) != "",
				InviteValue:    invite,
				OutputDir:      opts.output,
				JSON:           opts.jsonOutput,
				Quiet:          opts.quiet,
				NonInteractive: opts.nonInteractive,
				CollectBundle:  collectBundle,
				AllowRepair:    allowRepair,
				VerifyOnly:     verifyOnly,
				Yes:            yes,
			}})
		},
	}
	cmd.Flags().StringVar(&installerPath, "installer", "", "path to Grabber installer")
	cmd.Flags().StringVar(&invite, "invite", "", "Kickidler invite string")
	cmd.Flags().BoolVar(&collectBundle, "collect-bundle", false, "collect support bundle at the end")
	cmd.Flags().BoolVar(&allowRepair, "repair", false, "allow real repair after readiness checks and confirmation")
	cmd.Flags().BoolVar(&verifyOnly, "verify-only", false, "run read-only detection, recommendation, and verification only")
	cmd.Flags().BoolVar(&yes, "yes", false, "allow non-interactive repair when used with --repair")
	return cmd
}

func repairCommand(opts *globalOptions) *cobra.Command {
	var invite string
	var installerPath string
	var yes bool
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "repair",
		Short: "Repair Grabber by cleaning broken state, installing MSI, and ensuring Defender exclusions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun {
				return runWorkflowWithAdmin(cmd, opts, "repair --dry-run", false, repair.DryRunWorkflow{Invite: invite, Installer: installerPath, Yes: yes})
			}
			return runWorkflowWithAdmin(cmd, opts, "repair", true, repair.RepairWorkflow{Invite: invite, Installer: installerPath, Yes: yes})
		},
	}
	cmd.Flags().StringVar(&invite, "invite", "", "Kickidler invite string")
	cmd.Flags().StringVar(&installerPath, "installer", "", "path to Grabber installer")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm repair without prompting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview repair workflow without modifying the system")
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

func reportsCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reports",
		Short: "List and safely clean kigrepair report directories",
	}
	cmd.AddCommand(reportsListCommand(opts))
	cmd.AddCommand(reportsCleanupCommand(opts))
	return cmd
}

func reportsListCommand(opts *globalOptions) *cobra.Command {
	var limit int
	var all bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List kigrepair report directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := reportRootFromOptions(opts)
			result, err := reports.ListReports(root, reports.ReportListOptions{Limit: limit, All: all})
			exitCode := app.ExitSuccess
			if len(result.Warnings) > 0 {
				exitCode = app.ExitWarnings
			}
			if err != nil {
				exitCode = app.ExitUnexpectedError
			}
			if opts.jsonOutput {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				encoder.SetEscapeHTML(false)
				if encodeErr := encoder.Encode(result); encodeErr != nil {
					return encodeErr
				}
			} else if !opts.quiet {
				fmt.Fprint(cmd.OutOrStdout(), reports.FormatReportList(result))
			}
			if exitCode != app.ExitSuccess {
				return app.ExitError{Code: exitCode}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of reports to display")
	cmd.Flags().BoolVar(&all, "all", false, "show all reports when a limit is configured")
	return cmd
}

func reportsCleanupCommand(opts *globalOptions) *cobra.Command {
	var olderThan string
	var keepLast int
	var dryRun bool
	var yes bool
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Safely clean old kigrepair report directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReportWorkflow(opts, reports.CleanupReportsWorkflow{
				OlderThan: olderThan,
				KeepLast:  keepLast,
				DryRun:    dryRun,
				Yes:       yes,
			})
		},
	}
	cmd.Flags().StringVar(&olderThan, "older-than", reports.DefaultRetentionOlderThan, "delete reports older than this duration after keep-last is applied")
	cmd.Flags().IntVar(&keepLast, "keep-last", reports.DefaultRetentionKeepLast, "always keep the newest N report directories")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "create a cleanup plan without deleting reports")
	cmd.Flags().BoolVar(&yes, "yes", false, "execute the validated report cleanup plan")
	return cmd
}

func verifyCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verifies current Grabber installation state without modifying the system",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflowWithAdmin(cmd, opts, "verify", false, verifier.VerifyWorkflow{})
		},
	}
}

func preflightCommand(opts *globalOptions) *cobra.Command {
	var invite string
	var installerPath string
	cmd := &cobra.Command{
		Use:   "preflight",
		Short: "Checks whether this machine is ready for Grabber repair without modifying the system.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflowWithAdmin(cmd, opts, "preflight", false, preflight.Workflow{
				InstallerPath: installerPath,
				HasInvite:     strings.TrimSpace(invite) != "",
			})
		},
	}
	cmd.Flags().StringVar(&installerPath, "installer", "", "path to Grabber installer")
	cmd.Flags().StringVar(&invite, "invite", "", "Kickidler invite string")
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

func versionCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Get()
			if opts.jsonOutput {
				encoded, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(encoded))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", info.Tool, info.Version)
			fmt.Fprintf(cmd.OutOrStdout(), "Commit: %s\n", info.Commit)
			fmt.Fprintf(cmd.OutOrStdout(), "Build date: %s\n", info.BuildDate)
			fmt.Fprintf(cmd.OutOrStdout(), "Built by: %s\n", info.BuiltBy)
			fmt.Fprintf(cmd.OutOrStdout(), "Go: %s\n", info.GoVersion)
			fmt.Fprintf(cmd.OutOrStdout(), "Platform: %s/%s\n", info.OS, info.Arch)
			return nil
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

	suppressConsoleLog := ctx.Quiet || ctx.JSONOutput || workflow.Name() == "verify" || workflow.Name() == "preflight"
	logger, err := logging.New(reports.LogPath(ctx.OutputDir), suppressConsoleLog)
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
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(value); err != nil {
			return err
		}
	} else if !ctx.Quiet && workflow.Name() != "verify" && workflow.Name() != "repair --dry-run" && workflow.Name() != "preflight" {
		fmt.Printf("Report directory: %s\n", ctx.OutputDir)
	}
	if ctx.ExitCode != 0 {
		return app.ExitError{Code: ctx.ExitCode}
	}
	return nil
}

func runReportWorkflow(opts *globalOptions, workflow app.Workflow) error {
	ctx := app.NewContext()
	ctx.StartedAt = time.Now()
	ctx.ReportRoot = reportRootFromOptions(opts)
	ctx.OutputDir = uniqueTimestampedReportDir(ctx.ReportRoot, ctx.StartedAt)
	ctx.Quiet = opts.quiet
	ctx.NonInteractive = opts.nonInteractive
	ctx.Force = opts.force
	ctx.JSONOutput = opts.jsonOutput
	if opts.quiet {
		ctx.Mode = app.RunModeQuiet
	}

	reporter, err := reports.New(ctx.OutputDir)
	if err != nil {
		return err
	}
	ctx.Reporter = reporter

	suppressConsoleLog := ctx.Quiet || ctx.JSONOutput
	logger, err := logging.New(reports.LogPath(ctx.OutputDir), suppressConsoleLog)
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
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(value); err != nil {
			return err
		}
	} else if !ctx.Quiet {
		fmt.Printf("Report directory: %s\n", ctx.OutputDir)
	}
	if ctx.ExitCode != 0 {
		return app.ExitError{Code: ctx.ExitCode}
	}
	return nil
}

func reportRootFromOptions(opts *globalOptions) string {
	if strings.TrimSpace(opts.output) != "" {
		return opts.output
	}
	return config.DefaultReportRoot
}

func uniqueTimestampedReportDir(root string, startedAt time.Time) string {
	base := reports.TimestampedDir(root, startedAt)
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return base
	}
	for i := 1; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%02d", base, i)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d", base, startedAt.UnixNano())
}
