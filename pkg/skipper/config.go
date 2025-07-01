package skipper

// TODO: Add
// 1) Ability to change Is, As and other functions
// 2) Stringer
type Config struct {
	SkipTypes   map[string]pkgInfo `yaml:"skip_types"`
	WithDefault bool               `yaml:"with_default" env-default:"true"`
	Stringer    Stringer           `yaml:"stringer"`
}

type Stringer struct {
	FileName  string `yaml:"filename" env-default:"strings"`
	TagName   string `yaml:"tagname" env-default:"errgen"`
	Separator string `yaml:"separator" env-default:"\\n"`
	Connector string `yaml:"connector" env-default:": "`
}
