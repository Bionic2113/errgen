package collectr

import (
	"bytes"
	"go/parser"
	"go/printer"
	"go/token"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Bionic2113/errgen/internal/utils"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

// При записи в файл проверяем, что file != nil,
// Иначе нужно не обновлять, а создавать новый
type ErrorInfo struct {
	existsErrors map[string]string
}

type ErrorCollector struct {
	errorInfos map[utils.PkgInfo]*ErrorInfo
}

func New() (*ErrorCollector, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	ec := &ErrorCollector{errorInfos: make(map[utils.PkgInfo]*ErrorInfo)}
	if err := ec.ProcessFiles(currentDir); err != nil {
		return nil, err
	}

	return ec, nil
}

func (ec *ErrorCollector) ProcessFiles(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Нас интересуют только наши сгенерированные ошибки
		if !strings.HasSuffix(path, "error_gen.go") {
			return nil
		}

		return ec.ProcessFile(dir, path)
	})
}

func (ec *ErrorCollector) ProcessFile(dir, path string) error {
	node, err := decorator.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	pkgInfo := utils.PkgInfo{Name: node.Name.Name, Path: filepath.Dir(path)}
	ec.CollectErrors(node, pkgInfo, dir)

	return nil
}

func (ec *ErrorCollector) CollectErrors(node *dst.File, pkgInfo utils.PkgInfo, currentDir string) {
	dst.Inspect(node, func(n dst.Node) bool {
		gen, ok := n.(*dst.GenDecl)
		// ошибки создаются только в var, поэтому остальное отсекаем
		if !ok || gen.Tok != token.VAR {
			return true
		}

		for _, v := range gen.Specs {
			val, ok := v.(*dst.ValueSpec)
			// Страховка
			if !ok || len(val.Names) < 1 || len(val.Values) < 1 {
				continue
			}

			text, ok, _ := utils.ExtractErrorMessage(val.Values[0])
			if !ok {
				continue
			}

			einfo := ec.errorInfos[pkgInfo]
			if einfo == nil {
				einfo = &ErrorInfo{
					existsErrors: make(map[string]string),
				}
				ec.errorInfos[pkgInfo] = einfo
			}

			einfo.existsErrors[text] = val.Names[0].Name
		}

		return true
	})
}

func (ec *ErrorCollector) ErrorName(pkgInfo utils.PkgInfo, errText string) string {
	einfo := ec.errorInfos[pkgInfo]
	if einfo == nil {
		einfo = &ErrorInfo{
			existsErrors: make(map[string]string),
		}
		ec.errorInfos[pkgInfo] = einfo

		firstErr := generateErrorName(pkgInfo, "1")
		einfo.existsErrors[errText] = firstErr

		return firstErr
	}

	name, ok := einfo.existsErrors[errText]
	if ok {
		return name
	}

	nextErr := generateErrorName(pkgInfo, strconv.Itoa(len(einfo.existsErrors)+1))
	einfo.existsErrors[errText] = nextErr

	return nextErr
}

func generateErrorName(pkgInfo utils.PkgInfo, suffix string) string {
	return "Err" + strings.ToUpper(string(pkgInfo.Name[0])) + pkgInfo.Name[1:] + suffix
}

func (ec *ErrorCollector) GenerateFiles() error {
	for pkgInfo, einfo := range ec.errorInfos {
		if err := ec.generateFile(pkgInfo, einfo); err != nil {
			return err
		}
	}

	return nil
}

func (ec *ErrorCollector) generateFile(pkgInfo utils.PkgInfo, einfo *ErrorInfo) error {
	tmpl := `package {{.Package}}
import "errors"

var (
	{{range $text, $name := .Errors}}
	  {{$name}} = errors.New("{{$text}}") {{end}}
	)
`

	data := struct {
		Package string
		Errors  map[string]string
	}{Package: pkgInfo.Name, Errors: einfo.existsErrors}

	errFilePath := filepath.Join(pkgInfo.Path, "error_gen.go")

	f, err := os.Create(errFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	t, err := template.New("error_gen").Parse(tmpl)
	if err != nil {
		return err
	}

	formattedBuf := &bytes.Buffer{}
	if err := t.Execute(formattedBuf, data); err != nil {
		return err
	}

	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "", formattedBuf.String(), parser.ParseComments)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := cfg.Fprint(&buf, fset, astFile); err != nil {
		return err
	}

	if err := os.WriteFile(errFilePath, buf.Bytes(), 0o644); err != nil {
		return err
	}

	return nil
}
