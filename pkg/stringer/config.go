package stringer

import "github.com/Bionic2113/errgen/pkg/utils"

type Config struct {
	FileName  string `yaml:"filename" env-default:"strings"`
	TagName   string `yaml:"tagname" env-default:"errgen"`
	Separator string `yaml:"separator" env-default:"\\n"`
	Connector string `yaml:"connector" env-default:": "`
}

type Stringer struct {
	FileName    string
	TagName     string
	Separator   string
	Connector   string
	structsInfo map[utils.PkgInfo][]StructInfo
}

func NewStringer(cfg Config) *Stringer {
	return &Stringer{
		FileName:    cfg.FileName,
		TagName:     cfg.TagName,
		Separator:   cfg.Separator,
		Connector:   cfg.Connector,
		structsInfo: map[utils.PkgInfo][]StructInfo{},
	}
}

type StructInfo struct {
	Name   string
	Fields []*FieldInfo
}

type FieldInfo struct {
	FactName   string
	Type       string
	CustomName string
}
