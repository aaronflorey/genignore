package rulecatalog

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/aaronflorey/genignore/internal/signalfile"
)

type MatchResult struct {
	Matched bool
	Rule    Rule
	Path    string
	Line    int
}

func MatchEntry(root fs.FS, entry Entry) (bool, error) {
	result, err := MatchEntryWithResult(root, entry)
	return result.Matched, err
}

func MatchRule(root fs.FS, rule Rule) (bool, error) {
	result, err := MatchRuleWithResult(root, rule)
	return result.Matched, err
}

func MatchEntryWithResult(root fs.FS, entry Entry) (MatchResult, error) {
	for _, rule := range entry.Match {
		matched, err := MatchRuleWithResult(root, rule)
		if err != nil {
			return MatchResult{}, err
		}
		if matched.Matched {
			return matched, nil
		}
	}

	return MatchResult{}, nil
}

func MatchRuleWithResult(root fs.FS, rule Rule) (MatchResult, error) {
	switch rule.Type {
	case RuleTypeFilePath:
		matches, err := fs.Glob(root, rule.Path)
		if err != nil {
			return MatchResult{}, err
		}
		if len(matches) == 0 {
			return MatchResult{}, nil
		}
		return MatchResult{Matched: true, Rule: rule, Path: matches[0]}, nil
	case RuleTypeFileContentLine:
		content, err := signalfile.ReadFS(root, rule.Path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || signalfile.IsSafeSkip(err) {
				return MatchResult{}, nil
			}
			return MatchResult{}, err
		}
		line, matched := containsLine(content, rule.Contains)
		if !matched {
			return MatchResult{}, nil
		}
		return MatchResult{Matched: true, Rule: rule, Path: rule.Path, Line: line}, nil
	default:
		return MatchResult{}, nil
	}
}

func (r MatchResult) Evidence() string {
	if !r.Matched || r.Path == "" {
		return ""
	}
	if r.Line > 0 {
		return fmt.Sprintf("%s:%d", r.Path, r.Line)
	}
	return r.Path
}

func containsLine(content []byte, contains string) (int, bool) {
	for idx, line := range bytes.Split(content, []byte("\n")) {
		if strings.Contains(strings.TrimSuffix(string(line), "\r"), contains) {
			return idx + 1, true
		}
	}

	return 0, false
}
