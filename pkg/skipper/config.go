package skipper

type Config struct {
	SkipTypes   map[string]pkgInfo `yaml:"skip_types"`
	WithDefault bool               `yaml:"with_default" env-default:"true"`
}
