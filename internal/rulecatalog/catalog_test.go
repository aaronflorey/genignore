package rulecatalog

import (
	"encoding/json"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

type schemaDocument struct {
	ID          string                      `json:"$id"`
	Title       string                      `json:"title"`
	Description string                      `json:"description"`
	Required    []string                    `json:"required"`
	Properties  map[string]schemaProperty   `json:"properties"`
	Defs        map[string]schemaDefinition `json:"$defs"`
}

type schemaProperty struct {
	Type        string          `json:"type"`
	MinItems    int             `json:"minItems"`
	Ref         string          `json:"$ref"`
	Description string          `json:"description"`
	Items       *schemaProperty `json:"items"`
}

type schemaDefinition struct {
	Type                 string                    `json:"type"`
	AdditionalProperties *bool                     `json:"additionalProperties"`
	Required             []string                  `json:"required"`
	Properties           map[string]schemaProperty `json:"properties"`
	OneOf                []schemaProperty          `json:"oneOf"`
	Pattern              string                    `json:"pattern"`
	Description          string                    `json:"description"`
}

func TestInitError(t *testing.T) {
	t.Parallel()

	if err := InitError(); err != nil {
		t.Fatalf("InitError() = %v", err)
	}
}

func TestLoadEmbeddedCatalog(t *testing.T) {
	t.Parallel()

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("Load() returned no entries")
	}
	if !slices.IsSortedFunc(got, compareEntry) {
		t.Fatalf("Load() returned unsorted entries: %#v", got)
	}

	laravelIndex := slices.IndexFunc(got, func(entry Entry) bool {
		return entry.Provider == "laravel"
	})
	if laravelIndex < 0 {
		t.Fatalf("Load() missing laravel provider")
	}
	laravel := got[laravelIndex]
	if len(laravel.Match) != 2 {
		t.Fatalf("laravel match count = %d, want 2", len(laravel.Match))
	}
	if laravel.Match[1].Type != RuleTypeFileContentLine {
		t.Fatalf("laravel second rule type = %q, want %q", laravel.Match[1].Type, RuleTypeFileContentLine)
	}
	if laravel.Match[1].Contains != "laravel/framework" {
		t.Fatalf("laravel content evidence = %q, want %q", laravel.Match[1].Contains, "laravel/framework")
	}

	cloned := Entries()
	cloned[0].Provider = "mutated"
	if Entries()[0].Provider == "mutated" {
		t.Fatalf("Entries() should return a defensive copy")
	}
}

func TestRulesSchemaDocumentsGenignoreCatalogContract(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("rules.schema.json")
	if err != nil {
		t.Fatalf("ReadFile(rules.schema.json) = %v", err)
	}

	var doc schemaDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("json.Unmarshal(rules.schema.json) = %v", err)
	}

	if doc.ID != "https://genignore.dev/schemas/rules.schema.json" {
		t.Fatalf("schema $id = %q, want genignore-specific schema id", doc.ID)
	}
	if !strings.Contains(strings.ToLower(doc.Title), "genignore") {
		t.Fatalf("schema title = %q, want genignore-specific title", doc.Title)
	}
	if !strings.Contains(strings.ToLower(doc.Description), "genignore") {
		t.Fatalf("schema description = %q, want genignore-specific description", doc.Description)
	}
	if !slices.Equal(doc.Required, []string{"providers"}) {
		t.Fatalf("schema required = %#v, want only providers", doc.Required)
	}

	providers := doc.Properties["providers"]
	if providers.Type != "array" || providers.MinItems != 1 || providers.Items == nil || providers.Items.Ref != "#/$defs/provider_entry" {
		t.Fatalf("schema providers property = %#v, want non-empty provider_entry array", providers)
	}

	providerKey := doc.Defs["provider_key"]
	if !strings.Contains(strings.ToLower(providerKey.Description), "genignore provider key") {
		t.Fatalf("provider_key description = %q, want genignore provider key contract", providerKey.Description)
	}
	if providerKey.Pattern == "" {
		t.Fatal("provider_key pattern must be defined")
	}
	providerKeyPattern, err := regexp.Compile(providerKey.Pattern)
	if err != nil {
		t.Fatalf("regexp.Compile(provider_key.pattern) = %v", err)
	}
	for _, key := range []string{"global/al", "javascript/vue"} {
		if !providerKeyPattern.MatchString(key) {
			t.Fatalf("provider_key pattern %q rejected embedded provider key %q", providerKey.Pattern, key)
		}
	}
	for _, key := range []string{"/global/al", "global//al", "global/al/"} {
		if providerKeyPattern.MatchString(key) {
			t.Fatalf("provider_key pattern %q unexpectedly accepted invalid provider key %q", providerKey.Pattern, key)
		}
	}

	providerEntry := doc.Defs["provider_entry"]
	if providerEntry.Type != "object" || providerEntry.AdditionalProperties == nil || *providerEntry.AdditionalProperties {
		t.Fatalf("provider_entry = %#v, want object with additionalProperties=false", providerEntry)
	}
	if !slices.Equal(providerEntry.Required, []string{"provider", "match"}) {
		t.Fatalf("provider_entry required = %#v, want provider and match", providerEntry.Required)
	}

	matchRule := doc.Defs["match_rule"]
	if len(matchRule.OneOf) != 2 {
		t.Fatalf("match_rule oneOf count = %d, want 2", len(matchRule.OneOf))
	}
	if matchRule.OneOf[0].Ref != "#/$defs/file_path_rule" || matchRule.OneOf[1].Ref != "#/$defs/file_content_line_rule" {
		t.Fatalf("match_rule oneOf = %#v, want file_path_rule and file_content_line_rule", matchRule.OneOf)
	}

	filePathRule := doc.Defs["file_path_rule"]
	if !slices.Equal(filePathRule.Required, []string{"type", "path"}) {
		t.Fatalf("file_path_rule required = %#v, want type and path", filePathRule.Required)
	}

	fileContentLineRule := doc.Defs["file_content_line_rule"]
	if !slices.Equal(fileContentLineRule.Required, []string{"type", "path", "contains"}) {
		t.Fatalf("file_content_line_rule required = %#v, want type, path, contains", fileContentLineRule.Required)
	}
	if fileContentLineRule.Properties["contains"].Type != "string" {
		t.Fatalf("file_content_line_rule contains = %#v, want string", fileContentLineRule.Properties["contains"])
	}
	if !strings.Contains(strings.ToLower(fileContentLineRule.Properties["contains"].Description), "line") {
		t.Fatalf("file_content_line_rule contains description = %q, want line-based content contract", fileContentLineRule.Properties["contains"].Description)
	}
	if fileContentLineRule.Properties["path"].Ref != "#/$defs/relative_path" {
		t.Fatalf("file_content_line_rule path ref = %q, want relative_path", fileContentLineRule.Properties["path"].Ref)
	}
	if filePathRule.Properties["path"].Ref != "#/$defs/relative_path" {
		t.Fatalf("file_path_rule path ref = %q, want relative_path", filePathRule.Properties["path"].Ref)
	}

	relativePath := doc.Defs["relative_path"]
	if relativePath.Type != "string" || relativePath.Pattern == "" {
		t.Fatalf("relative_path = %#v, want constrained string path", relativePath)
	}
	if !strings.Contains(strings.ToLower(relativePath.Description), "project-relative") {
		t.Fatalf("relative_path description = %q, want project-relative path contract", relativePath.Description)
	}
}

func TestLoadFSAcceptsSlashDelimitedEmbeddedProviderKeys(t *testing.T) {
	t.Parallel()

	got, err := LoadFS(jsonFS(`
{
  "providers": [
    {
      "provider": " global/al ",
      "match": [
        {"type": "file_path", "path": ".alpackages"}
      ]
    },
    {
      "provider": "JavaScript/Vue",
      "match": [
        {"type": "file_path", "path": "package.json"}
      ]
    }
  ]
}
`), "rules.json")
	if err != nil {
		t.Fatalf("LoadFS() = %v", err)
	}

	want := []Entry{
		{Provider: "global/al", Match: []Rule{{Type: RuleTypeFilePath, Path: ".alpackages"}}},
		{Provider: "javascript/vue", Match: []Rule{{Type: RuleTypeFilePath, Path: "package.json"}}},
	}
	if !slices.EqualFunc(got, want, equalEntry) {
		t.Fatalf("LoadFS() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestLoadFSNormalizesAndSortsDeterministically(t *testing.T) {
	t.Parallel()

	got, err := LoadFS(jsonFS(`
{
  "providers": [
    {
      "provider": " Node ",
      "match": [
        {"type": "file_path", "path": "package.json"},
        {"type": "file_path", "path": "bun.lock"}
      ]
    },
    {
      "provider": "Go",
      "match": [
        {"type": "file_content_line", "path": "go.mod", "contains": "module example.com/demo"},
        {"type": "file_path", "path": "go.mod"}
      ]
    }
  ]
}
`), "rules.json")
	if err != nil {
		t.Fatalf("LoadFS() = %v", err)
	}

	want := []Entry{
		{
			Provider: "go",
			Match: []Rule{
				{Type: RuleTypeFileContentLine, Path: "go.mod", Contains: "module example.com/demo"},
				{Type: RuleTypeFilePath, Path: "go.mod"},
			},
		},
		{
			Provider: "node",
			Match: []Rule{
				{Type: RuleTypeFilePath, Path: "bun.lock"},
				{Type: RuleTypeFilePath, Path: "package.json"},
			},
		},
	}

	if !slices.EqualFunc(got, want, equalEntry) {
		t.Fatalf("LoadFS() mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestLoadFSRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	_, err := LoadFS(jsonFS(`
{
  "providers": [
    {
      "provider": "go",
      "match": [{"type": "file_path", "path": "go.mod"}],
      "extra": true
    }
  ]
}
`), "rules.json")
	if err == nil {
		t.Fatal("LoadFS() expected error")
	}
	if !strings.Contains(err.Error(), "unknown field \"extra\"") {
		t.Fatalf("LoadFS() error = %v, want unknown field failure", err)
	}
}

func TestLoadFSRejectsDuplicateProviderAfterNormalization(t *testing.T) {
	t.Parallel()

	_, err := LoadFS(jsonFS(`
{
  "providers": [
    {"provider": "Go", "match": [{"type": "file_path", "path": "go.mod"}]},
    {"provider": " go ", "match": [{"type": "file_path", "path": "main.go"}]}
  ]
}
`), "rules.json")
	if err == nil {
		t.Fatal("LoadFS() expected error")
	}
	if !strings.Contains(err.Error(), `duplicate provider "go"`) {
		t.Fatalf("LoadFS() error = %v, want duplicate provider failure", err)
	}
}

func TestLoadFSRejectsUnsupportedProvider(t *testing.T) {
	t.Parallel()

	_, err := LoadFS(jsonFS(`
{
  "providers": [
    {"provider": "unknown-provider", "match": [{"type": "file_path", "path": "go.mod"}]}
  ]
}
`), "rules.json")
	if err == nil {
		t.Fatal("LoadFS() expected error")
	}
	if !strings.Contains(err.Error(), `provider "unknown-provider" is not available in the embedded upstream or custom template catalogs`) {
		t.Fatalf("LoadFS() error = %v, want unsupported provider failure", err)
	}
}

func TestLoadFSRejectsDuplicateRule(t *testing.T) {
	t.Parallel()

	_, err := LoadFS(jsonFS(`
{
  "providers": [
    {
      "provider": "laravel",
      "match": [
        {"type": "file_content_line", "path": "composer.json", "contains": "laravel/framework"},
        {"type": "file_content_line", "path": "composer.json", "contains": "laravel/framework"}
      ]
    }
  ]
}
`), "rules.json")
	if err == nil {
		t.Fatal("LoadFS() expected error")
	}
	if !strings.Contains(err.Error(), `provider "laravel" contains duplicate rule`) {
		t.Fatalf("LoadFS() error = %v, want duplicate rule failure", err)
	}
}

func TestLoadFSRejectsEmptyFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "empty providers",
			json: `{"providers":[]}`,
			want: "providers must not be empty",
		},
		{
			name: "empty provider",
			json: `{"providers":[{"provider":"   ","match":[{"type":"file_path","path":"go.mod"}]}]}`,
			want: "provider 1 must not be empty",
		},
		{
			name: "empty match list",
			json: `{"providers":[{"provider":"go","match":[]}]}`,
			want: `provider "go" must define at least one match rule`,
		},
		{
			name: "empty rule path",
			json: `{"providers":[{"provider":"go","match":[{"type":"file_path","path":""}]}]}`,
			want: "path must not be empty",
		},
		{
			name: "empty rule contains",
			json: `{"providers":[{"provider":"laravel","match":[{"type":"file_content_line","path":"composer.json","contains":""}]}]}`,
			want: "file_content_line rules must define a non-empty contains value",
		},
	}

	for _, tt := range tests {
		ttt := tt
		t.Run(ttt.name, func(t *testing.T) {
			t.Parallel()

			_, err := LoadFS(jsonFS(ttt.json), "rules.json")
			if err == nil {
				t.Fatal("LoadFS() expected error")
			}
			if !strings.Contains(err.Error(), ttt.want) {
				t.Fatalf("LoadFS() error = %v, want substring %q", err, ttt.want)
			}
		})
	}
}

func jsonFS(content string) fs.FS {
	return fstest.MapFS{
		"rules.json": &fstest.MapFile{Data: []byte(strings.TrimSpace(content))},
	}
}

func equalEntry(a, b Entry) bool {
	return a.Provider == b.Provider && slices.Equal(a.Match, b.Match)
}
