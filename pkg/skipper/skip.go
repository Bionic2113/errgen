package skipper

import (
	"slices"

	"github.com/ilyakaznacheev/cleanenv"
)

type pkgInfo struct {
	All   bool
	Names []string
}

var banned = map[string]pkgInfo{
	"sync":         {All: true},
	"context":      {All: true},
	"database/sql": {Names: []string{"DB", "Conn", "Tx"}},
}
var workDir, module string

// TODO: Maybe it's better to remove init() and make a simple structure and constructor.
func init() {
	workDir, module = workDirAndModule()

	cfg := &Config{}
	if err := cleanenv.ReadConfig(".errgen.yaml", cfg); err != nil {
		println("[WARN] Not found config file (.errgen.yaml)\n", err.Error())
		return
	}

	if cfg.WithDefault {
		for k, v := range banned {
			info, ok := cfg.SkipTypes[k]
			if !ok {
				info.All = v.All
			}
			// Ignore dubplicates
			info.Names = append(info.Names, v.Names...)

			cfg.SkipTypes[k] = info
		}
	}

	banned = cfg.SkipTypes
}

func NeedSkip(name, path string) bool {
	info, ok := banned[path]
	if !ok {
		return false
	}

	if info.All {
		return true
	}

	return slices.Contains(info.Names, name)
}
