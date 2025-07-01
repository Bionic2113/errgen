package stringer

import (
	"bytes"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Bionic2113/errgen/pkg/utils"
)

const tmplt = `package {{.Package}}
import "fmt"

{{range .FuncsInfo}}
func (o {{.Owner}}) String() string {
	return fmt.Sprintf("{{.Return}}"{{range .Args}}, o.{{ . }}{{end}})
}
{{end}}
`

func (s *Stringer) GenerateFiles() error {
	for pkgInfo, structInfos := range s.structsInfo {
		if err := s.generateFile(pkgInfo, structInfos); err != nil {
			return err
		}
	}

	return nil
}

type funcInfo struct {
	Owner  string
	Return string
	Args   []string
}

func (s *Stringer) generateFile(pkgInfo utils.PkgInfo, structInfos []StructInfo) error {
	funcs := make([]funcInfo, len(structInfos))

	for i, si := range structInfos {
		parts, args := make([]string, len(si.Fields)), make([]string, len(si.Fields))
		for j, field := range si.Fields {
			name := field.FactName
			if field.CustomName != "" {
				name = field.CustomName
			}
			parts[j] = name + s.Connector + utils.Convert(field.Type)
			args[j] = field.FactName
		}

		funcs[i] = funcInfo{
			Owner:  si.Name,
			Return: strings.Join(parts, s.Separator),
			Args:   args,
		}
	}
	data := struct {
		Package   string
		FuncsInfo []funcInfo
	}{
		Package:   pkgInfo.Name,
		FuncsInfo: funcs,
	}
	errFilePath := filepath.Join(pkgInfo.Path, s.FileName+".go")

	f, err := os.Create(errFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	t, err := template.New("stringer").Parse(tmplt)
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
