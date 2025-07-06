package prcs

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bionic2113/errgen/internal/collector"
	"github.com/Bionic2113/errgen/internal/generator"
	"github.com/Bionic2113/errgen/pkg/utils"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type FileProcessor struct {
	packages          map[utils.PkgInfo][]utils.FunctionInfo
	currentDir        string
	collectorFilename string
	wrapperFilename   string
	collector         *collector.ErrorCollector
	stringer          Stringer
	skipper           Skipper
}

type Stringer interface {
	MakeStringFuncs(pkgInfo utils.PkgInfo, scope *dst.Scope)
	GenerateFiles() error
}

type Skipper interface {
	NeedSkipField(name, path string) bool
	ModuleName(path string) string
	NeedSkipFile(path string) bool
}

func New(
	collectorFilename string,
	wrapperFilename string,
	st Stringer,
	sk Skipper,
) (*FileProcessor, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	c, err := collector.New(collectorFilename)
	if err != nil {
		return nil, err
	}

	return &FileProcessor{
		currentDir:        currentDir,
		collectorFilename: collectorFilename,
		wrapperFilename:   wrapperFilename,
		packages:          make(map[utils.PkgInfo][]utils.FunctionInfo),
		collector:         c,
		stringer:          st,
		skipper:           sk,
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
			strings.HasSuffix(path, p.wrapperFilename+".go") ||
			strings.HasSuffix(path, p.collectorFilename+".go") ||
			p.skipper.NeedSkipFile(path) {
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

	p.stringer.MakeStringFuncs(pkgInfo, node.Scope)

	subPkg := utils.SubPackageName(pkgInfo.Path, p.currentDir)
	fileName := filepath.Base(path)
	functions := generator.AnalyzeFunctions(
		node, pkgInfo, subPkg,
		p.currentDir, fileName, p.collector,
		p.skipper,
	)
	if len(functions) > 0 {
		p.packages[pkgInfo] = append(p.packages[pkgInfo], functions...)
	}

	return nil
}

func (p *FileProcessor) GenerateErrorFiles() {
	if err := p.collector.GenerateFiles(); err != nil {
		panic("[FileProcessor] - GenerateErrorFiles - collector.GenerateFiles: " + err.Error())
	}

	if err := p.stringer.GenerateFiles(); err != nil {
		panic("[FileProcessor] - GenerateErrorFiles - stringer.GenerateFiles: " + err.Error())
	}

	for pkg, functions := range p.packages {
		generator.GenerateErrorFile(p.wrapperFilename, pkg, functions)
	}
}
