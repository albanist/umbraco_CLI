package validate

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var (
	controlChars = regexp.MustCompile(`[\x00-\x1F\x7F]`)
	preEncoded   = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)
)

func Path(value string) error {
	normalized := strings.ReplaceAll(value, "\\", "/")
	if strings.HasPrefix(normalized, "/") {
		return fmt.Errorf("invalid path: %s", value)
	}
	if len(normalized) >= 2 && (normalized[1] == ':' && ((normalized[0] >= 'A' && normalized[0] <= 'Z') || (normalized[0] >= 'a' && normalized[0] <= 'z'))) {
		return fmt.Errorf("invalid path: %s", value)
	}
	if path.Clean(normalized) == ".." || strings.Contains(normalized, "../") || strings.Contains(normalized, "/..") {
		return fmt.Errorf("invalid path: %s", value)
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return fmt.Errorf("invalid path: %s", value)
		}
	}
	return nil
}

func String(value string) error {
	if controlChars.MatchString(value) {
		return fmt.Errorf("input contains control characters")
	}
	return nil
}

func ResourceID(value string) error {
	if strings.ContainsAny(value, "?#%") {
		return fmt.Errorf("invalid resource ID: %s", value)
	}
	return nil
}

func NoPreEncoding(value string) error {
	if preEncoded.MatchString(value) {
		return fmt.Errorf("input appears to be pre-encoded")
	}
	return nil
}

func ValidateValue(value any, keyHint string) error {
	switch v := value.(type) {
	case string:
		if err := String(v); err != nil {
			return err
		}
		if err := NoPreEncoding(v); err != nil {
			return err
		}
		hint := strings.ToLower(keyHint)
		if strings.Contains(hint, "id") {
			if err := ResourceID(v); err != nil {
				return err
			}
		}
		if strings.Contains(hint, "path") {
			if err := Path(v); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range v {
			if err := ValidateValue(item, keyHint); err != nil {
				return err
			}
		}
	case map[string]any:
		for key, nested := range v {
			if err := ValidateValue(nested, key); err != nil {
				return err
			}
		}
	}
	return nil
}

func Input(input map[string]any) error {
	for key, value := range input {
		if err := ValidateValue(value, key); err != nil {
			return err
		}
	}
	return nil
}
