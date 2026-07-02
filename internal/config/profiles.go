package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultProfileName = "default"

type UserConfigSelection struct {
	Profile    string
	Path       string
	CustomPath bool
	Active     bool
	Explicit   bool
}

type UserProfile struct {
	Name      string
	Path      string
	Config    Config
	Active    bool
	IsDefault bool
}

func ProfileConfigPath(profile string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return profileConfigPath(homeDir, profile)
}

func ActiveProfile() (string, bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}
	return readActiveProfile(homeDir)
}

func SetActiveProfile(profile string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	normalized, err := normalizeProfileName(profile)
	if err != nil {
		return err
	}
	path := activeProfilePath(homeDir)
	if normalized == "" || normalized == defaultProfileName {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(normalized+"\n"), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func (s UserConfigSelection) IsDefault() bool {
	return s.Profile == "" || s.Profile == defaultProfileName
}

func selectedConfigPath(workingDir string, homeDir string, opts LoadOptions) (UserConfigSelection, bool, error) {
	if strings.TrimSpace(opts.ConfigPath) != "" {
		path, err := expandConfigPath(opts.ConfigPath, workingDir, homeDir)
		if err != nil {
			return UserConfigSelection{}, false, err
		}
		return UserConfigSelection{
			Profile:    "",
			Path:       path,
			CustomPath: true,
			Explicit:   true,
		}, true, nil
	}

	if strings.TrimSpace(opts.Profile) != "" {
		path, err := profileConfigPath(homeDir, opts.Profile)
		if err != nil {
			return UserConfigSelection{}, false, err
		}
		profile, _ := normalizeProfileName(opts.Profile)
		return UserConfigSelection{
			Profile:  profile,
			Path:     path,
			Explicit: true,
		}, true, nil
	}

	active, ok, err := readActiveProfile(homeDir)
	if err != nil {
		return UserConfigSelection{}, false, err
	}
	if ok {
		path, err := profileConfigPath(homeDir, active)
		if err != nil {
			return UserConfigSelection{}, false, err
		}
		return UserConfigSelection{
			Profile: active,
			Path:    path,
			Active:  true,
		}, true, nil
	}

	return UserConfigSelection{}, false, nil
}

func expandConfigPath(rawPath string, workingDir string, homeDir string) (string, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return "", fmt.Errorf("config path cannot be empty")
	}
	if strings.HasPrefix(path, "~/") {
		if homeDir == "" {
			return "", fmt.Errorf("cannot expand %q without a home directory", rawPath)
		}
		path = filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
	}
	if !filepath.IsAbs(path) {
		base := workingDir
		if base == "" {
			var err error
			base, err = os.Getwd()
			if err != nil {
				return "", err
			}
		}
		path = filepath.Join(base, path)
	}
	return filepath.Clean(path), nil
}

func profileConfigPath(homeDir string, profile string) (string, error) {
	normalized, err := normalizeProfileName(profile)
	if err != nil {
		return "", err
	}
	if homeDir == "" {
		return "", fmt.Errorf("cannot resolve profile %q without a home directory", normalized)
	}
	if normalized == "" || normalized == defaultProfileName {
		return filepath.Join(homeDir, ".umbraco", "config.json"), nil
	}
	return filepath.Join(homeDir, ".umbraco", normalized+".config.json"), nil
}

func normalizeProfileName(profile string) (string, error) {
	value := strings.TrimSpace(profile)
	if value == "" {
		return "", nil
	}
	if value == defaultProfileName {
		return defaultProfileName, nil
	}
	if strings.Contains(value, "..") {
		return "", fmt.Errorf("invalid profile name %q: profile names cannot contain '..'", profile)
	}
	for _, r := range value {
		valid := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' ||
			r == '_' ||
			r == '.'
		if !valid {
			return "", fmt.Errorf("invalid profile name %q: use letters, numbers, '.', '-' or '_'", profile)
		}
	}
	return value, nil
}

func activeProfilePath(homeDir string) string {
	return filepath.Join(homeDir, ".umbraco", "active-profile")
}

func readActiveProfile(homeDir string) (string, bool, error) {
	if homeDir == "" {
		return "", false, nil
	}
	payload, err := os.ReadFile(activeProfilePath(homeDir))
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	profile, err := normalizeProfileName(string(payload))
	if err != nil {
		return "", false, err
	}
	if profile == "" || profile == defaultProfileName {
		return "", false, nil
	}
	return profile, true, nil
}

func ListUserProfiles() ([]UserProfile, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(homeDir, ".umbraco")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	active, hasActive, err := readActiveProfile(homeDir)
	if err != nil {
		return nil, err
	}

	profiles := make([]UserProfile, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		profileName := ""
		isDefault := false
		switch {
		case name == "config.json":
			profileName = defaultProfileName
			isDefault = true
		case strings.HasSuffix(name, ".config.json"):
			profileName = strings.TrimSuffix(name, ".config.json")
		default:
			continue
		}
		if _, err := normalizeProfileName(profileName); err != nil {
			continue
		}
		path := filepath.Join(dir, name)
		cfg, ok, err := loadUserConfigAtPath(path)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		profiles = append(profiles, UserProfile{
			Name:      profileName,
			Path:      path,
			Config:    cfg,
			Active:    (hasActive && active == profileName) || (!hasActive && isDefault),
			IsDefault: isDefault,
		})
	}

	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].IsDefault != profiles[j].IsDefault {
			return profiles[i].IsDefault
		}
		return profiles[i].Name < profiles[j].Name
	})
	return profiles, nil
}
