package main

import (
	"github.com/Bionic2113/errgen/internal/prcs"
)

func main() {
	processor, err := prcs.New()
	if err != nil {
		panic(err)
	}
	if err := processor.ProcessFiles(); err != nil {
		panic(err)
	}
	processor.GenerateErrorFiles()
}
