package rulecatalog

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"

	"github.com/aaronflorey/genignore/internal/customtemplate"
	"github.com/aaronflorey/genignore/internal/templatecatalog"
)

//go:embed rules.json
var rulesFS embed.FS

type RuleType string

const (
	RuleTypeFilePath        RuleType = "file_path"
	RuleTypeFileContentLine RuleType = "file_content_line"
)

type Entry struct {
	Provider string `json:"provider"`
	Match    []Rule `json:"match"`
}

type Rule struct {
	Type     RuleType `json:"type"`
	Path     string   `json:"path"`
	Contains string   `json:"contains,omitempty"`
}

type rawCatalog struct {
	Schema    string     `json:"$schema,omitempty"`
	Providers []rawEntry `json:"providers"`
}

type rawEntry struct {
	Provider string    `json:"provider"`
	Match    []rawRule `json:"match"`
}

type rawRule struct {
	Type     string  `json:"type"`
	Path     string  `json:"path"`
	Contains *string `json:"contains,omitempty"`
}

var (
	entries []Entry
	initErr error
)

func init() {
	entries, initErr = Load()
}

func InitError() error {
	return initErr
}

func Entries() []Entry {
	return cloneEntries(entries)
}

func Load() ([]Entry, error) {
	return LoadFS(rulesFS, "rules.json")
}

func LoadFS(root fs.FS, path string) ([]Entry, error) {
	raw, err := fs.ReadFile(root, path)
	if err != nil {
		return nil, fmt.Errorf("read rule catalog %q: %w", path, err)
	}

	entries, err := loadBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("decode rule catalog %q: %w", path, err)
	}

	return entries, nil
}

func loadBytes(raw []byte) ([]Entry, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	var catalog rawCatalog
	if err := decoder.Decode(&catalog); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); err == nil {
		return nil, fmt.Errorf("expected a single JSON object")
	} else if err != io.EOF {
		return nil, err
	}

	if len(catalog.Providers) == 0 {
		return nil, fmt.Errorf("providers must not be empty")
	}

	entries := make([]Entry, 0, len(catalog.Providers))
	seenProviders := make(map[string]struct{}, len(catalog.Providers))
	for entryIndex, rawEntry := range catalog.Providers {
		provider := normalizeProvider(rawEntry.Provider)
		if provider == "" {
			return nil, fmt.Errorf("provider %d must not be empty", entryIndex+1)
		}
		if err := validateProvider(provider); err != nil {
			return nil, err
		}
		if len(rawEntry.Match) == 0 {
			return nil, fmt.Errorf("provider %q must define at least one match rule", provider)
		}
		if _, exists := seenProviders[provider]; exists {
			return nil, fmt.Errorf("duplicate provider %q", provider)
		}
		seenProviders[provider] = struct{}{}

		rules := make([]Rule, 0, len(rawEntry.Match))
		seenRules := make(map[string]struct{}, len(rawEntry.Match))
		for ruleIndex, rawRule := range rawEntry.Match {
			rule, err := normalizeRule(rawRule)
			if err != nil {
				return nil, fmt.Errorf("provider %q rule %d: %w", provider, ruleIndex+1, err)
			}

			key := ruleKey(rule)
			if _, exists := seenRules[key]; exists {
				return nil, fmt.Errorf("provider %q contains duplicate rule %q", provider, key)
			}
			seenRules[key] = struct{}{}
			rules = append(rules, rule)
		}

		slices.SortFunc(rules, compareRule)
		entries = append(entries, Entry{Provider: provider, Match: rules})
	}

	slices.SortFunc(entries, compareEntry)
	return entries, nil
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func validateProvider(provider string) error {
	if err := templatecatalog.InitError(); err != nil {
		return fmt.Errorf("load embedded upstream template catalog: %w", err)
	}
	if err := customtemplate.InitError(); err != nil {
		return fmt.Errorf("load embedded custom template registry: %w", err)
	}
	if templatecatalog.HasProvider(provider) || customtemplate.HasProvider(provider) {
		return nil
	}
	return fmt.Errorf("provider %q is not available in the embedded upstream or custom template catalogs", provider)
}

func normalizeRule(rawRule rawRule) (Rule, error) {
	if strings.TrimSpace(rawRule.Path) == "" {
		return Rule{}, fmt.Errorf("path must not be empty")
	}

	switch RuleType(rawRule.Type) {
	case RuleTypeFilePath:
		if rawRule.Contains != nil {
			return Rule{}, fmt.Errorf("file_path rules must not set contains")
		}
		return Rule{Type: RuleTypeFilePath, Path: rawRule.Path}, nil
	case RuleTypeFileContentLine:
		if rawRule.Contains == nil || strings.TrimSpace(*rawRule.Contains) == "" {
			return Rule{}, fmt.Errorf("file_content_line rules must define a non-empty contains value")
		}
		return Rule{Type: RuleTypeFileContentLine, Path: rawRule.Path, Contains: *rawRule.Contains}, nil
	default:
		if strings.TrimSpace(rawRule.Type) == "" {
			return Rule{}, fmt.Errorf("type must not be empty")
		}
		return Rule{}, fmt.Errorf("unsupported rule type %q", rawRule.Type)
	}
}

func compareEntry(a, b Entry) int {
	return strings.Compare(a.Provider, b.Provider)
}

func compareRule(a, b Rule) int {
	if cmp := strings.Compare(a.Path, b.Path); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(string(a.Type), string(b.Type)); cmp != 0 {
		return cmp
	}
	return strings.Compare(a.Contains, b.Contains)
}

func ruleKey(rule Rule) string {
	return strings.Join([]string{string(rule.Type), rule.Path, rule.Contains}, "\x00")
}

func cloneEntries(src []Entry) []Entry {
	if len(src) == 0 {
		return nil
	}

	cloned := make([]Entry, len(src))
	for i, entry := range src {
		cloned[i] = Entry{
			Provider: entry.Provider,
			Match:    slices.Clone(entry.Match),
		}
	}
	return cloned
}
