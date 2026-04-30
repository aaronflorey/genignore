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
	Detect(ctx context.Context, opts DetectOptions) (CommandResult, error)
	Add(ctx context.Context, opts AddOptions) (CommandResult, error)
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
	detectCmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect providers and rebuild managed block",
		RunE: func(_ *cobra.Command, _ []string) error {
			res, runErr := service.Detect(context.Background(), DetectOptions{
				Include: include,
				Exclude: exclude,
				DryRun:  dryRun,
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

	var addDryRun bool
	addCmd := &cobra.Command{
		Use:   "add <keys...>",
		Short: "Add providers to existing managed set",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, keys []string) error {
			res, runErr := service.Add(context.Background(), AddOptions{
				Keys:    keys,
				DryRun:  addDryRun,
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

	root.AddCommand(detectCmd, addCmd, listCmd, searchCmd)
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
