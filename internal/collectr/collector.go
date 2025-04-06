package collectr

import (
	"go/parser"
	"go/token"
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
	file         *dst.File
	existsErrors map[string]string
	newErrors    map[string]string
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
					file:         node,
					existsErrors: make(map[string]string),
					newErrors:    make(map[string]string),
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
			newErrors:    make(map[string]string),
		}
		ec.errorInfos[pkgInfo] = einfo

		firstErr := generateErrorName(pkgInfo, "1")
		einfo.newErrors[errText] = firstErr

		return firstErr
	}

	name, ok := einfo.existsErrors[errText]
	if ok {
		return name
	}

	name, ok = einfo.newErrors[errText]
	if ok {
		return name
	}

	nextErr := generateErrorName(pkgInfo, strconv.Itoa(len(einfo.existsErrors)+len(einfo.newErrors)))
	einfo.newErrors[errText] = nextErr

	return nextErr
}

func generateErrorName(pkgInfo utils.PkgInfo, suffix string) string {
	return "Err" + strings.ToUpper(string(pkgInfo.Name[0])) + pkgInfo.Name[1:] + suffix
}

func (ec *ErrorCollector) GenerateFiles() error {
	for pkgInfo, einfo := range ec.errorInfos {
		if einfo.file == nil {
			if err := ec.generateFile(pkgInfo, einfo); err != nil {
				return err
			}

			continue
		}

		// TODO: А точно ли стоит делать апдейт, мб просто такая же генарация и всё
		if err := ec.updateFile(pkgInfo.Path, einfo); err != nil {
			return err
		}
	}

	return nil
}

// TODO: дописать
func (ec *ErrorCollector) generateFile(pkgInfo utils.PkgInfo, einfo *ErrorInfo) error {
	return nil
}

// TODO: дописать
func (ec *ErrorCollector) updateFile(path string, einfo *ErrorInfo) error {
	path = filepath.Join(path, "error.go")
	return nil
}
