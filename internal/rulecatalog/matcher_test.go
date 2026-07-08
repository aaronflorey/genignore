package rulecatalog

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestMatchEntryUsesORSemanticsAcrossRules(t *testing.T) {
	t.Parallel()

	entry := Entry{
		Provider: "jetbrains",
		Match: []Rule{
			{Type: RuleTypeFilePath, Path: ".idea"},
			{Type: RuleTypeFilePath, Path: "*.iml"},
		},
	}

	matched, err := MatchEntry(fstest.MapFS{
		"project.iml": &fstest.MapFile{Data: []byte("module")},
	}, entry)
	if err != nil {
		t.Fatalf("MatchEntry() error = %v", err)
	}
	if !matched {
		t.Fatal("MatchEntry() = false, want true when any rule matches")
	}
}

func TestMatchEntryReturnsFalseWhenNoRulesMatch(t *testing.T) {
	t.Parallel()

	entry := Entry{
		Provider: "laravel",
		Match: []Rule{
			{Type: RuleTypeFilePath, Path: "artisan"},
			{Type: RuleTypeFileContentLine, Path: "composer.json", Contains: "laravel/framework"},
		},
	}

	matched, err := MatchEntry(fstest.MapFS{
		"composer.json": &fstest.MapFile{Data: []byte("{\n  \"require\": {\n    \"symfony/console\": \"^7.0\"\n  }\n}\n")},
	}, entry)
	if err != nil {
		t.Fatalf("MatchEntry() error = %v", err)
	}
	if matched {
		t.Fatal("MatchEntry() = true, want false when no rules match")
	}
}

func TestMatchRuleMatchesContentWithinSingleLineOfFileBody(t *testing.T) {
	t.Parallel()

	matched, err := MatchRule(fstest.MapFS{
		"composer.json": &fstest.MapFile{Data: []byte("{\r\n  \"require\": {\r\n    \"laravel/framework\": \"^11.0\"\r\n  }\r\n}\r\n")},
	}, Rule{Type: RuleTypeFileContentLine, Path: "composer.json", Contains: "laravel/framework"})
	if err != nil {
		t.Fatalf("MatchRule() error = %v", err)
	}
	if !matched {
		t.Fatal("MatchRule() = false, want true when a file body line contains the substring")
	}
}

func TestMatchRuleDoesNotCrossLineBoundariesForContentMatch(t *testing.T) {
	t.Parallel()

	matched, err := MatchRule(fstest.MapFS{
		"composer.json": &fstest.MapFile{Data: []byte("laravel/\nframework\n")},
	}, Rule{Type: RuleTypeFileContentLine, Path: "composer.json", Contains: "laravel/framework"})
	if err != nil {
		t.Fatalf("MatchRule() error = %v", err)
	}
	if matched {
		t.Fatal("MatchRule() = true, want false when the substring only appears across multiple lines")
	}
}

func TestMatchRuleMatchesExactDirectoryPath(t *testing.T) {
	t.Parallel()

	matched, err := MatchRule(fstest.MapFS{
		".idea": &fstest.MapFile{Mode: fs.ModeDir},
	}, Rule{Type: RuleTypeFilePath, Path: ".idea"})
	if err != nil {
		t.Fatalf("MatchRule() error = %v", err)
	}
	if !matched {
		t.Fatal("MatchRule() = false, want true for an exact directory path match")
	}
}

func TestMatchRuleWithResultSurfacesStablePathEvidence(t *testing.T) {
	t.Parallel()

	result, err := MatchRuleWithResult(fstest.MapFS{
		"package.json": &fstest.MapFile{Data: []byte(`{"name":"demo"}`)},
	}, Rule{Type: RuleTypeFilePath, Path: "package.json"})
	if err != nil {
		t.Fatalf("MatchRuleWithResult() error = %v", err)
	}

	want := MatchResult{Matched: true, Rule: Rule{Type: RuleTypeFilePath, Path: "package.json"}, Path: "package.json"}
	if result != want {
		t.Fatalf("MatchRuleWithResult() = %+v, want %+v", result, want)
	}
	if evidence := result.Evidence(); evidence != "package.json" {
		t.Fatalf("Evidence() = %q, want %q", evidence, "package.json")
	}
}

func TestMatchEntryWithResultSurfacesStableContentLineEvidence(t *testing.T) {
	t.Parallel()

	entry := Entry{
		Provider: "laravel",
		Match: []Rule{
			{Type: RuleTypeFilePath, Path: "artisan"},
			{Type: RuleTypeFileContentLine, Path: "composer.json", Contains: "laravel/framework"},
		},
	}

	result, err := MatchEntryWithResult(fstest.MapFS{
		"composer.json": &fstest.MapFile{Data: []byte("{\n  \"require\": {\n    \"laravel/framework\": \"^11.0\"\n  }\n}\n")},
	}, entry)
	if err != nil {
		t.Fatalf("MatchEntryWithResult() error = %v", err)
	}

	want := MatchResult{
		Matched: true,
		Rule:    Rule{Type: RuleTypeFileContentLine, Path: "composer.json", Contains: "laravel/framework"},
		Path:    "composer.json",
		Line:    3,
	}
	if result != want {
		t.Fatalf("MatchEntryWithResult() = %+v, want %+v", result, want)
	}
	if evidence := result.Evidence(); evidence != "composer.json:3" {
		t.Fatalf("Evidence() = %q, want %q", evidence, "composer.json:3")
	}
}
