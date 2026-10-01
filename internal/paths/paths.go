package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandHome replaces leading ~ with home directory, as it's not done by shell e.g. in configuration file
func ExpandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}
