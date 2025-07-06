package main

import (
	"github.com/Bionic2113/errgen/internal/prcs"
	"github.com/Bionic2113/errgen/pkg/skipper"
	"github.com/Bionic2113/errgen/pkg/stringer"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Skipper           skipper.Config  `yaml:"skipper"`
	Stringer          stringer.Config `yaml:"stringer"`
	WrapperFilename   string          `yaml:"wrapper_filename"`
	SimpleErrFilename string          `yaml:"simple_err_filename"`
}

func main() {
	cfg := &Config{}
	if err := cleanenv.ReadConfig(".errgen.yaml", cfg); err != nil {
		panic("[WARN] Not found config file (.errgen.yaml): " + err.Error())
	}

	processor, err := prcs.New(
		cfg.SimpleErrFilename, cfg.WrapperFilename,
		stringer.NewStringer(cfg.Stringer),
		skipper.New(cfg.Skipper),
	)

	if err != nil {
		panic(err)
	}
	if err := processor.ProcessFiles(); err != nil {
		panic(err)
	}

	processor.GenerateErrorFiles()
}
