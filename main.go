package main

import (
	"github.com/Bionic2113/errgen/internal/prcs"
	"github.com/Bionic2113/errgen/pkg/skipper"
	"github.com/Bionic2113/errgen/pkg/stringer"
)

func main() {
	stCfg := skipper.StringerConfig()
	processor, err := prcs.New(
		stringer.NewStringer(stCfg.FileName, stCfg.TagName, stCfg.Separator, stCfg.Connector),
	)
	if err != nil {
		panic(err)
	}
	if err := processor.ProcessFiles(); err != nil {
		panic(err)
	}
	processor.GenerateErrorFiles()
}
