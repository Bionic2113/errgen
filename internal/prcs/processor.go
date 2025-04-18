package prcs

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bionic2113/errgen/internal/collectr"
	"github.com/Bionic2113/errgen/internal/utils"
	"github.com/dave/dst/decorator"
)

type FileProcessor struct {
	packages   map[utils.PkgInfo][]utils.FunctionInfo
	currentDir string
	collector  *collectr.ErrorCollector
}

func New() (*FileProcessor, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	c, err := collectr.New()
	if err != nil {
		return nil, err
	}

	return &FileProcessor{
		currentDir: currentDir,
		packages:   make(map[utils.PkgInfo][]utils.FunctionInfo),
		collector:  c,
	}, nil
}

func (p *FileProcessor) ProcessFiles() error {
	return filepath.Walk(p.currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем директории, тесты, файлы с ошибками и main.go
		if info.IsDir() ||
			!strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") ||
			strings.HasSuffix(path, "_mock.go") ||
			strings.HasSuffix(path, ".pb.go") ||
			strings.HasSuffix(path, "errors.go") ||
			strings.HasSuffix(path, "error_gen.go") ||
			strings.HasSuffix(path, "main.go") {
			return nil
		}

		return p.ProcessFile(path)
	})
}

func (p *FileProcessor) ProcessFile(path string) error {
	fset := token.NewFileSet()
	node, err := decorator.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	pkgInfo := utils.PkgInfo{Name: node.Name.Name, Path: filepath.Dir(path)}
	// пропустим моки
	if strings.HasSuffix(pkgInfo.Path, "mocks") {
		return nil
	}

	subPkg := utils.SubPackageName(pkgInfo.Path, p.currentDir)
	fileName := filepath.Base(path)
	functions := utils.AnalyzeFunctions(node, pkgInfo, subPkg, p.currentDir, fileName, p.collector)
	if len(functions) > 0 {
		p.packages[pkgInfo] = append(p.packages[pkgInfo], functions...)
	}

	return nil
}

func (p *FileProcessor) GenerateErrorFiles() {
	if err := p.collector.GenerateFiles(); err != nil {
		panic("[FileProcessor] - GenerateErrorFiles - collector.GenerateFiles: " + err.Error())
	}
	for pkg, functions := range p.packages {
		utils.GenerateErrorFile(pkg, functions)
	}
}
