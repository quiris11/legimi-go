package account

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPasswordIsReadBackUnchangedOrNotSaved(t *testing.T) {
	passwords := []string{
		"abc", "a#b", "a;b", "a=b", "a`b", `a"b`, "a'b", ` ab `, "a\\b", "a\\", "zażółć",
		`a"""b`, "a`b\"\"\"c", "#ab", `"ab`, `ab"`, `"ab"`, `'ab'`,
	}
	for _, password := range passwords {
		configFile := filepath.Join(t.TempDir(), "config.ini")
		repository := newFileAccountRepository(configFile)
		repository.SaveLogin("user@example.com")
		saveErr := repository.SavePassword(password)

		reloaded := newFileAccountRepository(configFile)
		if reloaded.GetLogin() != "user@example.com" {
			t.Errorf("password %q: login read back as %q", password, reloaded.GetLogin())
		}
		got := reloaded.GetPassword()
		if saveErr == nil && got != password {
			t.Errorf("password %q: read back as %q", password, got)
		}
		if saveErr != nil && got != "" {
			t.Errorf("password %q: not saved (%v), but read back as %q", password, saveErr, got)
		}
	}
}

func TestConfigFileIsReadableOnlyByOwner(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "dir", "config.ini")
	newFileAccountRepository(configFile).SaveLogin("user@example.com")
	for path, mode := range map[string]os.FileMode{filepath.Dir(configFile): 0700, configFile: 0600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != mode {
			t.Errorf("%s: mode %o, expected %o", path, info.Mode().Perm(), mode)
		}
	}
}
