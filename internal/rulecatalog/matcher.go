package rulecatalog

import (
	"bytes"
	"errors"
	"io/fs"
	"strings"
)

func MatchEntry(root fs.FS, entry Entry) (bool, error) {
	for _, rule := range entry.Match {
		matched, err := MatchRule(root, rule)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}

	return false, nil
}

func MatchRule(root fs.FS, rule Rule) (bool, error) {
	switch rule.Type {
	case RuleTypeFilePath:
		matches, err := fs.Glob(root, rule.Path)
		if err != nil {
			return false, err
		}
		return len(matches) > 0, nil
	case RuleTypeFileContentLine:
		content, err := fs.ReadFile(root, rule.Path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return false, nil
			}
			return false, err
		}
		return containsLine(content, rule.Contains), nil
	default:
		return false, nil
	}
}

func containsLine(content []byte, contains string) bool {
	for _, line := range bytes.Split(content, []byte("\n")) {
		if strings.Contains(strings.TrimSuffix(string(line), "\r"), contains) {
			return true
		}
	}

	return false
}
