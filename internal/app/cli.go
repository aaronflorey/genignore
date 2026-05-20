package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/aaronflorey/genignore/internal/api"
	"github.com/aaronflorey/genignore/internal/provider"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type commandService interface {
	Resolve(ctx context.Context, opts ResolveOptions) (ResolveResult, error)
	Detect(ctx context.Context, opts DetectOptions) (CommandResult, error)
	Add(ctx context.Context, opts AddOptions) (CommandResult, error)
	Doctor(ctx context.Context, opts DoctorOptions) (DoctorResult, error)
}

var newCommandService = func(cwd string, cfg Config) commandService {
	return NewService(cwd, cfg)
}

var newCatalogClient = func() providerCatalog {
	return api.NewClient()
}

func Run(args []string) int {
	if err := runtimeInitError(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		return 1
	}

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	service := newCommandService(cwd, cfg)
	catalogClient := newCatalogClient()
	root := &cobra.Command{
		Use:   "genignore",
		Short: "Generate and manage gitignore block",
	}

	jsonOutput := false
	verbose := false
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output machine-readable JSON")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "show verbose detection info")

	var include, exclude []string
	var dryRun bool
	var diff bool
	resolveCmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve providers without mutating .gitignore",
		RunE: func(_ *cobra.Command, _ []string) error {
			res, runErr := service.Resolve(context.Background(), ResolveOptions{
				Include: include,
				Exclude: exclude,
				Verbose: verbose,
			})
			if runErr != nil {
				return runErr
			}
			printResolveResult(res, jsonOutput, verbose)
			return nil
		},
	}
	resolveCmd.Flags().StringSliceVar(&include, "include", nil, "provider keys to include")
	resolveCmd.Flags().StringSliceVar(&exclude, "exclude", nil, "provider keys to exclude")

	detectCmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect providers and rebuild managed block",
		RunE: func(_ *cobra.Command, _ []string) error {
			res, runErr := service.Detect(context.Background(), DetectOptions{
				Include: include,
				Exclude: exclude,
				DryRun:  dryRun,
				Diff:    diff,
				Verbose: verbose,
			})
			if runErr != nil {
				return runErr
			}
			printResult(res, jsonOutput, verbose)
			return nil
		},
	}
	detectCmd.Flags().StringSliceVar(&include, "include", nil, "provider keys to include")
	detectCmd.Flags().StringSliceVar(&exclude, "exclude", nil, "provider keys to exclude")
	detectCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would change without writing files")
	detectCmd.Flags().BoolVar(&diff, "diff", false, "show the exact managed-block diff without writing files")

	var addDryRun bool
	var addDiff bool
	addCmd := &cobra.Command{
		Use:   "add <keys...>",
		Short: "Add providers to existing managed set",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, keys []string) error {
			res, runErr := service.Add(context.Background(), AddOptions{
				Keys:    keys,
				DryRun:  addDryRun,
				Diff:    addDiff,
				Verbose: verbose,
			})
			if runErr != nil {
				return runErr
			}
			printResult(res, jsonOutput, verbose)
			return nil
		},
	}
	addCmd.Flags().BoolVar(&addDryRun, "dry-run", false, "show what would change without writing files")
	addCmd.Flags().BoolVar(&addDiff, "diff", false, "show the exact managed-block diff without writing files")

	var doctorInclude, doctorExclude []string
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Explain detection, provider resolution, and runtime decisions",
		RunE: func(_ *cobra.Command, _ []string) error {
			res, runErr := service.Doctor(context.Background(), DoctorOptions{Include: doctorInclude, Exclude: doctorExclude})
			if runErr != nil {
				return runErr
			}
			printDoctorResult(res, jsonOutput)
			return nil
		},
	}
	doctorCmd.Flags().StringSliceVar(&doctorInclude, "include", nil, "provider keys to include")
	doctorCmd.Flags().StringSliceVar(&doctorExclude, "exclude", nil, "provider keys to exclude")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all supported provider keys",
		RunE: func(_ *cobra.Command, _ []string) error {
			providers, err := ListProviders(context.Background(), catalogClient)
			if err != nil {
				return err
			}
			printCatalogResult(CatalogResult{
				Command:   "list",
				Providers: providers,
			}, jsonOutput)
			return nil
		},
	}

	searchCmd := &cobra.Command{
		Use:   "search <term>",
		Short: "Search supported provider keys",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			term := args[0]
			providers, err := SearchProviders(context.Background(), catalogClient, term)
			if err != nil {
				return err
			}
			printCatalogResult(CatalogResult{
				Command:   "search",
				Query:     term,
				Providers: providers,
			}, jsonOutput)
			return nil
		},
	}

	root.AddCommand(resolveCmd, detectCmd, addCmd, doctorCmd, listCmd, searchCmd)
	root.SetArgs(args)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

type CatalogResult struct {
	Command   string   `json:"command"`
	Query     string   `json:"query,omitempty"`
	Providers []string `json:"providers"`
}

func printCatalogResult(result CatalogResult, jsonOutput bool) {
	if jsonOutput {
		bytes, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(bytes))
		return
	}
	label := lipgloss.NewStyle().Bold(true)
	fmt.Printf("%s %s\n", label.Render("Command:"), result.Command)
	if result.Query != "" {
		fmt.Printf("%s %s\n", label.Render("Query:"), result.Query)
	}
	fmt.Printf("%s\n", label.Render("Providers:"))
	for _, key := range result.Providers {
		fmt.Println(key)
	}
}

func printResolveResult(result ResolveResult, jsonOutput bool, verbose bool) {
	if jsonOutput {
		bytes, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(bytes))
		return
	}
	label := lipgloss.NewStyle().Bold(true)
	fmt.Printf("%s %s\n", label.Render("Command:"), result.Command)
	if len(result.DetectedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Detected:"), formatProviderList(result.DetectedProviders))
	}
	if len(result.IncludedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Included:"), formatProviderList(result.IncludedProviders))
	}
	if len(result.ExcludedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Excluded:"), formatProviderList(result.ExcludedProviders))
	}
	fmt.Printf("%s %s\n", label.Render("Final:"), formatProviderList(result.FinalProviders))
	for _, warning := range result.UnsupportedKeyWarnings {
		fmt.Printf("%s %s\n", label.Render("Warning:"), warning)
	}
	if verbose {
		for _, detection := range result.DetectionResults {
			if !detection.Matched && detection.Error == "" {
				continue
			}
			fmt.Printf("%s %s\n", label.Render("Detection:"), formatDetectionResult(detection))
		}
	}
}

func printResult(result CommandResult, jsonOutput bool, verbose bool) {
	if jsonOutput {
		bytes, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(bytes))
		return
	}
	label := lipgloss.NewStyle().Bold(true)
	fmt.Printf("%s %s\n", label.Render("Command:"), result.Command)
	if len(result.Targets) > 0 {
		for _, target := range result.Targets {
			fmt.Printf("%s %s\n", label.Render("Target:"), target.Path)
			if len(target.DetectedProviders) > 0 {
				fmt.Printf("%s %s\n", label.Render("Detected:"), formatProviderList(target.DetectedProviders))
			}
			if len(target.FinalProviders) > 0 {
				fmt.Printf("%s %s\n", label.Render("Final:"), formatProviderList(target.FinalProviders))
			}
			if verbose {
				for _, detection := range target.DetectionResults {
					if !detection.Matched && detection.Error == "" {
						continue
					}
					fmt.Printf("%s %s\n", label.Render("Detection:"), formatDetectionResult(detection))
				}
			}
			if target.FileAction != "" {
				fmt.Printf("%s %s\n", label.Render("File:"), target.FileAction)
			}
		}
		fmt.Printf("%s %s\n", label.Render("Final:"), formatProviderList(result.FinalProviders))
		for _, warning := range result.UnsupportedKeyWarnings {
			fmt.Printf("%s %s\n", label.Render("Warning:"), warning)
		}
		for _, warning := range result.RuntimeWarnings {
			fmt.Printf("%s %s\n", label.Render("Warning:"), warning)
		}
		for _, warning := range result.RemoteProviderWarnings {
			fmt.Printf("%s %s\n", label.Render("Warning:"), warning)
		}
		if result.FileAction != "" {
			fmt.Printf("%s %s\n", label.Render("File:"), result.FileAction)
		}
		return
	}
	if len(result.DetectedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Detected:"), formatProviderList(result.DetectedProviders))
	}
	if len(result.AddedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Added:"), formatProviderList(result.AddedProviders))
	}
	if len(result.IncludedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Included:"), formatProviderList(result.IncludedProviders))
	}
	if len(result.ExcludedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Excluded:"), formatProviderList(result.ExcludedProviders))
	}
	fmt.Printf("%s %s\n", label.Render("Final:"), formatProviderList(result.FinalProviders))
	for _, warning := range result.UnsupportedKeyWarnings {
		fmt.Printf("%s %s\n", label.Render("Warning:"), warning)
	}
	for _, warning := range result.RuntimeWarnings {
		fmt.Printf("%s %s\n", label.Render("Warning:"), warning)
	}
	for _, warning := range result.RemoteProviderWarnings {
		fmt.Printf("%s %s\n", label.Render("Warning:"), warning)
	}
	if verbose {
		for _, detection := range result.DetectionResults {
			if !detection.Matched && detection.Error == "" {
				continue
			}
			fmt.Printf("%s %s\n", label.Render("Detection:"), formatDetectionResult(detection))
		}
	}
	if result.FileAction != "" {
		fmt.Printf("%s %s\n", label.Render("File:"), result.FileAction)
	}
	if result.PreviewOnly {
		fmt.Printf("%s %s\n", label.Render("Preview:"), "diff-only (no file written)")
	}
	if result.PreviewOnly && result.Diff != "" {
		fmt.Printf("%s\n%s\n", label.Render("Diff:"), result.Diff)
	}
}

func printDoctorResult(result DoctorResult, jsonOutput bool) {
	if jsonOutput {
		bytes, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(bytes))
		return
	}
	label := lipgloss.NewStyle().Bold(true)
	fmt.Printf("%s %s\n", label.Render("Command:"), result.Command)
	if len(result.DetectedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Detected:"), formatProviderList(result.DetectedProviders))
	}
	if len(result.IncludedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Included:"), formatProviderList(result.IncludedProviders))
	}
	if len(result.ExcludedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Excluded:"), formatProviderList(result.ExcludedProviders))
	}
	fmt.Printf("%s %s\n", label.Render("Final:"), formatProviderList(result.FinalProviders))
	for _, warning := range result.UnsupportedKeyWarnings {
		fmt.Printf("%s %s\n", label.Render("Warning:"), warning)
	}
	for _, warning := range result.RuntimeWarnings {
		fmt.Printf("%s %s\n", label.Render("Warning:"), warning)
	}
	for _, warning := range result.RemoteProviderWarnings {
		fmt.Printf("%s %s\n", label.Render("Warning:"), warning)
	}
	for _, detection := range result.Detections {
		if !detection.Matched && detection.Error == "" {
			continue
		}
		fmt.Printf("%s %s\n", label.Render("Detection:"), formatDoctorDetection(detection))
	}
	fmt.Printf("%s %t\n", label.Render("Offline:"), result.Runtime.Offline)
	fmt.Printf("%s %s\n", label.Render("Upstream:"), result.Runtime.UpstreamCommit)
	if len(result.Runtime.RemoteProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Remote:"), formatProviderList(result.Runtime.RemoteProviders))
	}
	if len(result.Runtime.EmbeddedProviders) > 0 {
		fmt.Printf("%s %s\n", label.Render("Embedded:"), formatProviderList(result.Runtime.EmbeddedProviders))
	}
	for _, entry := range result.Runtime.CacheEntries {
		fmt.Printf("%s %s\n", label.Render("Cache:"), formatDoctorCacheEntry(entry))
	}
	for _, decision := range result.Runtime.Decisions {
		fmt.Printf("%s %s\n", label.Render("Decision:"), decision)
	}
	for _, line := range result.Provenance {
		fmt.Printf("%s %s\n", label.Render("Provenance:"), strings.TrimPrefix(line, "# Provenance: "))
	}
}

func formatProviderList(providers []string) string {
	if len(providers) == 0 {
		return ""
	}

	sorted := append([]string(nil), providers...)
	slices.Sort(sorted)
	return strings.Join(sorted, ", ")
}

func formatDetectionResult(result provider.Result) string {
	status := "skipped"
	if result.Matched {
		status = "matched"
	} else if result.Error != "" {
		status = "error"
	}

	parts := []string{result.Key, status, result.Reason}
	if result.Evidence != "" {
		parts = append(parts, result.Evidence)
	}
	if result.Error != "" {
		parts = append(parts, result.Error)
	}

	return strings.Join(parts, " | ")
}

func formatDoctorDetection(result DoctorDetection) string {
	status := "skipped"
	if result.Matched {
		status = "matched"
	} else if result.Error != "" {
		status = "error"
	}
	parts := []string{result.Key, result.Origin, status, result.Reason}
	if result.Evidence != "" {
		parts = append(parts, result.Evidence)
	}
	if result.Error != "" {
		parts = append(parts, result.Error)
	}
	return strings.Join(parts, " | ")
}

func formatDoctorCacheEntry(entry DoctorCacheEntry) string {
	parts := []string{entry.Provider, entry.State}
	if entry.Detail != "" {
		parts = append(parts, entry.Detail)
	}
	return strings.Join(parts, " | ")
}
