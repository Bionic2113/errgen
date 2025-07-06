package skipper

import (
	"log/slog"
	"os"
)

// TODO(bionic2113): Add
// 1) Ability to change Is, As and other functions
type Config struct {
	SkipTypes   map[string]pkgInfo `yaml:"skip_types"`
	WithDefault bool               `yaml:"with_default" env-default:"true"`
	Rules       []Rule             `yaml:"rules"`
}

type Skipper struct {
	Config
	workDir string
	module  string
	l       *slog.Logger
}

func New(cfg Config) *Skipper {
	sk := &Skipper{
		Config: cfg,
		l:      slog.New(slog.NewJSONHandler(os.Stdout, nil)).WithGroup("Skipper"),
	}

	sk.workDirAndModule()

	if !sk.WithDefault {
		sk.l.Info("Without default preset")
		return sk
	}

	for k, v := range defaultSkipTypes {
		info, ok := sk.SkipTypes[k]
		if !ok {
			info.All = v.All
		}
		// Ignore dubplicates
		info.Names = append(info.Names, v.Names...)

		sk.SkipTypes[k] = info
	}

	sk.Rules = append(sk.Rules, defaultRules...)

	return sk
}
