package assets

import (
	"os"
	"path/filepath"

	"github.com/kujtimiihoxha/termai/internal/config"
)

func WriteAssets() error {
	appCfg := config.Get()
	appWd := config.WorkingDirectory()
	scriptDir := filepath.Join(
		appWd,
		appCfg.Data.Directory,
		"diff",
	)
	scriptPath := filepath.Join(scriptDir, "index.mjs")
	// Before, run the script in cmd/diff/main.go to build this file
	if _, err := os.Stat(scriptPath); err != nil {
		scriptData, err := FS.ReadFile("diff/index.mjs")
		if err != nil {
			return err
		}

		err = os.MkdirAll(scriptDir, 0o755)
		if err != nil {
			return err
		}
		err = os.WriteFile(scriptPath, scriptData, 0o755)
		if err != nil {
			return err
		}
	}

	themeDir := filepath.Join(
		appWd,
		appCfg.Data.Directory,
		"themes",
	)

	themePath := filepath.Join(themeDir, "dark.json")

	if _, err := os.Stat(themePath); err != nil {
		themeData, err := FS.ReadFile("diff/themes/dark.json")
		if err != nil {
			return err
		}

		err = os.MkdirAll(themeDir, 0o755)
		if err != nil {
			return err
		}
		err = os.WriteFile(themePath, themeData, 0o755)
		if err != nil {
			return err
		}
	}
	return nil
}
