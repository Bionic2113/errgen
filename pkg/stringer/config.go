package stringer

import "github.com/Bionic2113/errgen/pkg/utils"

type Stringer struct {
	FileName    string
	TagName     string
	Separator   string
	Connector   string
	structsInfo map[utils.PkgInfo][]StructInfo
}

func NewStringer(fileName, tagName, separator, connector string) *Stringer {
	return &Stringer{
		FileName:    fileName,
		TagName:     tagName,
		Separator:   separator,
		Connector:   connector,
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
