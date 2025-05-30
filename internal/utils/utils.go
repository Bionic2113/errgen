package utils

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Bionic2113/errgen/pkg/skipper"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

var (
	versionMatch         *regexp.Regexp = regexp.MustCompile(`^(/?\w+(\.?|-?))+(/v\d+)$`)
	prefixAndSuffixMatch *regexp.Regexp = regexp.MustCompile(`(go-\w+)|(\w+-go)`)
)

type ErrorInformator interface {
	ErrorName(pkgInfo PkgInfo, errText string) string
}

func SubPackageName(pkgDir, baseDir string) string {
	rel, err := filepath.Rel(baseDir, pkgDir)
	if err != nil {
		return ""
	}

	if rel == "." {
		return ""
	}

	return rel
}

func AnalyzeFunctions(node *dst.File, pkgInfo PkgInfo, subPkg, currentDir, fileName string, ei ErrorInformator) []FunctionInfo {
	var functions []FunctionInfo
	imports := CollectImports(node)
	originalPath := filepath.Join(currentDir, subPkg, fileName)

	dst.Inspect(node, func(n dst.Node) bool {
		if funcDecl, ok := n.(*dst.FuncDecl); ok && HasErrorReturn(funcDecl) {
			f := CreateFunctionInfo(funcDecl, pkgInfo, subPkg, imports)
			functions = append(functions, f)
			ModifyFunctionBody(funcDecl, f, pkgInfo, ei)

		}
		return true
	})

	if len(functions) > 0 {
		RemoveUnusedImports(node)
		WriteModifiedFile(node, originalPath)
	}

	return functions
}

func CollectImports(node *dst.File) map[string]Path {
	imports := make(map[string]Path)
	for _, imp := range node.Imports {
		if imp.Path == nil {
			continue
		}
		path := strings.Trim(imp.Path.Value, `"`)

		if imp.Name != nil {
			imports[imp.Name.Name] = Path{Alias: imp.Name.Name, Path: path}
			continue
		}

		imports[NameFromPath(path)] = Path{Path: path}
	}

	return imports
}

func NameFromPath(path string) string {
	// Удаляем версии пакетов, если есть
	matches := versionMatch.FindStringSubmatch(path)
	if len(matches) > 0 {
		path = strings.ReplaceAll(path, matches[len(matches)-1], "")
	}

	name := filepath.Base(path)

	// Удаляем префикс или суффикс go
	matches = prefixAndSuffixMatch.FindStringSubmatch(name)
	if len(matches) == 0 {
		return name
	}

	if matches[1] == "" {
		return strings.TrimSuffix(name, "-go")
	}

	return strings.TrimPrefix(name, "go-")
}

func CreateFunctionInfo(funcDecl *dst.FuncDecl, pkgInfo PkgInfo, subPkg string, imports map[string]Path) FunctionInfo {
	args := ExtractArgs(funcDecl, pkgInfo.Path, imports)
	receiverType := ExtractReceiverType(funcDecl)

	return FunctionInfo{PackageName: pkgInfo.Name, SubPackageName: subPkg, FunctionName: funcDecl.Name.Name, ReceiverType: receiverType, Args: args, Imports: imports, HasError: true}
}

func ExtractReceiverType(funcDecl *dst.FuncDecl) string {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return ""
	}

	if starExpr, ok := funcDecl.Recv.List[0].Type.(*dst.StarExpr); ok {
		if ident, ok := starExpr.X.(*dst.Ident); ok {
			return ident.Name
		}
	} else if ident, ok := funcDecl.Recv.List[0].Type.(*dst.Ident); ok {
		return ident.Name
	}

	return ""
}

func WriteModifiedFile(node *dst.File, path string) {
	var buf bytes.Buffer
	if err := decorator.Fprint(&buf, node); err != nil {
		fmt.Printf("Error formatting modified file: %v\n", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		fmt.Printf("Error writing modified file: %v\n", err)
	}
}

func HasErrorReturn(funcDecl *dst.FuncDecl) bool {
	if funcDecl.Type.Results == nil {
		return false
	}

	for _, result := range funcDecl.Type.Results.List {
		if ident, ok := result.Type.(*dst.Ident); ok {
			if ident.Name == "error" {
				return true
			}
		}
	}

	return false
}

func ErrorReturnIndex(funcDecl *dst.FuncDecl) int {
	if funcDecl.Type.Results == nil {
		return -1
	}

	var totalIndex int
	for _, result := range funcDecl.Type.Results.List {
		if ident, ok := result.Type.(*dst.Ident); ok {
			if ident.Name == "error" {
				return totalIndex
			}
		}
		totalIndex++
	}
	return -1
}

func ExtractArgs(funcDecl *dst.FuncDecl, path string, imports map[string]Path) []ArgInfo {
	var args []ArgInfo
	if funcDecl.Type.Params == nil {
		return args
	}

	for _, field := range funcDecl.Type.Params.List {
		var typeStr string
		expr := field.Type
		if v, ok := expr.(*dst.ArrayType); ok {
			typeStr = "[]"
			expr = v.Elt
		}

		if v, ok := expr.(*dst.StarExpr); ok {
			typeStr += "*"
			expr = v.X
		}

		var isSelector bool
		if v, ok := expr.(*dst.SelectorExpr); ok {
			pkg := v.X.(*dst.Ident).Name
			if skipper.NeedSkip(v.Sel.Name, imports[pkg].Path) {
				continue
			}
			typeStr += pkg + "."
			expr = v.Sel
			isSelector = true
		}

		if v, ok := expr.(*dst.Ident); ok {
			if !isSelector && skipper.NeedSkip(v.Name, skipper.ModuleName(path)) {
				continue
			}
			typeStr += v.Name
		} else {
			typeStr += "any"
		}

		for _, name := range field.Names {
			args = append(args, ArgInfo{Name: name.Name, Type: typeStr})
		}
	}
	return args
}

func GenerateErrorFile(pkgInfo PkgInfo, functions []FunctionInfo) {
	imports := map[string]Path{"errors": {Path: "errors"}}
	for _, f := range functions {
		for _, arg := range f.Args {

			switch {
			case strings.HasPrefix(arg.Type, "time."):
				imports["time"] = Path{Path: "time"}

			case arg.Type == "int" || arg.Type == "int64" || arg.Type == "uint64":
				imports["strconv"] = Path{Path: "strconv"}
			case arg.Type == "float64":
				imports["strconv"] = Path{Path: "strconv"}

			case arg.Type == "bool":
				imports["strconv"] = Path{Path: "strconv"}
			case arg.Type == "any" || strings.Contains(arg.Type, "[]"):
				imports["fmt"] = Path{Path: "fmt"}
			case !IsBasicType(arg.Type):
				imports["fmt"] = Path{Path: "fmt"}
			}
			if strings.Contains(arg.Type, ".") {
				parts := strings.SplitN(arg.Type, ".", 2)
				if len(parts) == 2 {
					pkgName := strings.TrimPrefix(strings.TrimPrefix(parts[0], "[]"), "*")
					if importPath, ok := f.Imports[pkgName]; ok {
						imports[pkgName] = importPath
					}
				}
			}
		}
	}

	templateData := ErrorTemplate{Package: pkgInfo.Name, Functions: functions}
	tmpl := `package {{.Package}}

import (
	{{range $k, $val := .Imports}} {{$val.Alias}} "{{$val.Path}}"
{{end}})
{{range .Functions}}

type {{.FunctionName}}Error struct {
	{{- range .Args}}
	{{.Name}} {{.Type}}
	{{- end}}
	reason string
	err    error
}

func New{{.FunctionName}}Error({{range .Args}}{{.Name}} {{.Type}}, {{end}}reason string, err error) *{{.FunctionName}}Error {
	return &{{.FunctionName}}Error{
		{{- range .Args}}
		{{.Name}}: {{.Name}},
		{{- end}}
		reason: reason,
		err:    err,
	}
}

func (e *{{.FunctionName}}Error) Error() string {
	return "[" +
		{{if .SubPackageName}}"{{.SubPackageName}}/" +{{end}}
		"{{.PackageName}}" +
		{{if .ReceiverType}}".{{.ReceiverType}}" +{{end}}
		"] - " +
		"{{.FunctionName}} - " +
		e.reason +
		{{if .Args}}
		" - args: {" +
		{{range $i, $arg := .Args}}{{if $i}} + ", " +{{end}}
		"{{.Name}}: " + {{if eq .Type "string"}}e.{{.Name}}{{else if eq .Type "int"}}strconv.Itoa(e.{{.Name}}){{else if eq .Type "int64"}}strconv.FormatInt(e.{{.Name}}, 10){{else if eq .Type "uint64"}}strconv.FormatUint(e.{{.Name}}, 10){{else if eq .Type "float64"}}strconv.FormatFloat(e.{{.Name}}, 'f', -1, 64){{else if eq .Type "bool"}}strconv.FormatBool(e.{{.Name}}){{else}}fmt.Sprintf("%#v", e.{{.Name}}){{end}}{{end}} +
		"}" +
		{{end}}
		"\n" +
		e.err.Error()
}

func (e *{{.FunctionName}}Error) Unwrap() error {
	return e.err
}

func (e *{{.FunctionName}}Error) Is(target error) bool {
	if _, ok := target.(*{{.FunctionName}}Error); ok {
		return true
	}
	return errors.Is(e.err, target)
}
{{end}}`

	data := struct {
		Package   string
		Functions []FunctionInfo
		Imports   map[string]Path
	}{Package: templateData.Package, Functions: templateData.Functions, Imports: imports}

	errFilePath := filepath.Join(pkgInfo.Path, "errors.go")

	f, err := os.Create(errFilePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	t, err := template.New("errors").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	formattedBuf := &bytes.Buffer{}
	if err := t.Execute(formattedBuf, data); err != nil {
		panic(err)
	}

	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "", formattedBuf.String(), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	if err := cfg.Fprint(&buf, fset, astFile); err != nil {
		panic(err)
	}

	if err := os.WriteFile(errFilePath, buf.Bytes(), 0o644); err != nil {
		panic(err)
	}
}

func IsBasicType(typeName string) bool {
	basicTypes := map[string]bool{"string": true, "int": true, "int64": true, "uint64": true, "float64": true, "bool": true, "any": true}
	return basicTypes[strings.TrimPrefix(typeName, "*")]
}

func ModifyFunctionBody(funcDecl *dst.FuncDecl, info FunctionInfo, pkgInfo PkgInfo, ei ErrorInformator) {
	parentMap := make(map[dst.Node]dst.Node)
	dst.Inspect(funcDecl.Body, func(n dst.Node) bool {
		if n == nil {
			return false
		}
		dst.Inspect(n, func(child dst.Node) bool {
			if child == nil {
				return false
			}
			if child != n {
				parentMap[child] = n
			}
			return true
		})
		return true
	})

	errorIndex := ErrorReturnIndex(funcDecl)
	if errorIndex == -1 {
		return
	}

	dst.Inspect(funcDecl.Body, func(n dst.Node) bool {
		returnStmt, ok := n.(*dst.ReturnStmt)
		if !ok || errorIndex >= len(returnStmt.Results) {
			return true
		}

		result := returnStmt.Results[errorIndex]
		if IsNilError(result) {
			return true
		}

		if !IsNeedChange(result) {
			return true
		}

		// Определяем сообщение об ошибке и нужно ли использовать nil
		reason := "unknown error in " + info.FunctionName

		// Проверяем не создается ли ошибка напрямую
		msg, ok, useNilError := ExtractErrorMessage(result)
		if !ok {
			// Проверяем, не является ли ошибка результатом вызова функции
			var funcLit bool
			msg, ok, funcLit = FindLastFunctionCall(returnStmt, parentMap)
			if funcLit {
				return true
			}
		}

		if msg != "" {
			reason = msg
		}

		errArg := IsErrorWrapper(result)
		if errArg == nil {
			errArg = result
			if useNilError {
				errArg = dst.NewIdent(ei.ErrorName(pkgInfo, reason))
				reason = "unknown error in " + info.FunctionName
			}
		}

		constructorCall := &dst.CallExpr{
			Fun: dst.NewIdent("New" + info.FunctionName + "Error"),
			Args: append(
				ArgumentNames(funcDecl, info.Args),
				dst.NewIdent("\""+reason+"\""),
				errArg,
			),
		}
		returnStmt.Results[errorIndex] = constructorCall

		return true
	})
}

func IsNilError(expr dst.Expr) bool {
	if ident, ok := expr.(*dst.Ident); ok {
		return ident.Name == "nil"
	}

	return false
}

// Проверяем нужно ли нам заменить возврат ошибки
// на обертку
func IsNeedChange(expr dst.Expr) bool {
	call, ok := expr.(*dst.CallExpr)
	if !ok {
		return true
	}

	// Если не наша обертка, то пропускаем
	if ident, ok := call.Fun.(*dst.Ident); ok {
		return !strings.HasSuffix(ident.Name, "Error")
	}

	// Если эти функции - это создание через fmt или errors,
	// то обрабатываем
	if selector, ok := call.Fun.(*dst.SelectorExpr); ok {
		if ident, ok := selector.X.(*dst.Ident); ok {
			return ident.Name == "fmt" || ident.Name == "errors"
		}
	}

	return false
}

func ArgumentNames(funcDecl *dst.FuncDecl, args []ArgInfo) []dst.Expr {
	result := make([]dst.Expr, len(args))
	for i, v := range args {
		result[i] = dst.NewIdent(v.Name)
	}

	return result
}

func IsErrorWrapper(expr dst.Expr) dst.Expr {
	if callExpr, ok := expr.(*dst.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*dst.Ident); ok {
			if strings.HasSuffix(ident.Name, "Error") {
				return callExpr.Args[len(callExpr.Args)-1]
			}
		}
	}

	return nil
}

func RemoveUnusedImports(node *dst.File) {
	imports := make(map[string]*dst.ImportSpec)

	for _, imp := range node.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			imports[path] = imp
		}
	}
	dst.Inspect(node, func(n dst.Node) bool {
		if callExpr, ok := n.(*dst.CallExpr); ok {
			n = callExpr.Fun
		}
		if sel, ok := n.(*dst.SelectorExpr); ok {
			if ident, ok := sel.X.(*dst.Ident); ok {
				for path, imp := range imports {
					pkgName := ""
					if imp.Name != nil {
						pkgName = imp.Name.Name
					} else {
						pkgName = NameFromPath(path)
					}
					if ident.Name == pkgName {
						delete(imports, path)
					}
				}
			}
		}

		return true
	})

	for _, imp := range node.Decls {
		if genDecl, ok := imp.(*dst.GenDecl); ok && genDecl.Tok == token.IMPORT {
			var newImports []dst.Spec
			for _, spec := range genDecl.Specs {
				if importSpec, ok := spec.(*dst.ImportSpec); ok {
					path := strings.Trim(importSpec.Path.Value, `"`)
					if _, unused := imports[path]; !unused {
						newImports = append(newImports, importSpec)
					}
				}
			}
			genDecl.Specs = newImports
		}
	}

	// Мало ли у кого-то в нескольких импортах находится,
	// поэтому снова в цикле проходим и пропускаем пустые
	var newDecls []dst.Decl
	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok && genDecl.Tok == token.IMPORT {
			if len(genDecl.Specs) > 0 {
				newDecls = append(newDecls, decl)
			}
			continue
		}
		newDecls = append(newDecls, decl)
	}

	node.Decls = newDecls
}

// TODO: refactor
func ExtractErrorMessage(expr dst.Expr) (string, bool, bool) {
	callExpr, ok := expr.(*dst.CallExpr)
	if !ok {
		return "", false, false
	}
	if sel, ok := callExpr.Fun.(*dst.SelectorExpr); ok {
		if ident, ok := sel.X.(*dst.Ident); ok {
			if (ident.Name == "errors" && sel.Sel.Name == "New") || (ident.Name == "fmt" && (sel.Sel.Name == "Errorf")) {
				if len(callExpr.Args) > 0 {
					if lit, ok := callExpr.Args[0].(*dst.BasicLit); ok {
						return strings.Trim(lit.Value, `"`), true, true
					}

					var buf bytes.Buffer
					printer.Fprint(&buf, token.NewFileSet(), callExpr.Args[0])

					return buf.String(), true, true
				}
			}

			return ident.Name + "." + sel.Sel.Name, true, false
		}

		return sel.Sel.Name, true, false
	} else if ident, ok := callExpr.Fun.(*dst.Ident); ok {
		// Если уже была обертка, то забираем причину
		if strings.HasSuffix(ident.Name, "Error") && len(callExpr.Args) > 1 {
			if lit, ok := callExpr.Args[len(callExpr.Args)-2].(*dst.BasicLit); ok {
				return strings.Trim(lit.Value, `"`), true, false
			}
		}

		return ident.Name, true, false
	}

	return "", false, false
}

func FindLastFunctionCall(node dst.Node, parentMap map[dst.Node]dst.Node) (string, bool, bool) {
	parent := parentMap[node]
	for parent != nil {
		var assignStmt *dst.AssignStmt
		switch stmt := parent.(type) {
		default:
			parent = parentMap[parent]
			continue
		case *dst.FuncLit:
			return "", false, true
		case *dst.AssignStmt:
			assignStmt = stmt
		case *dst.IfStmt: // TODO: refactor
			a, ok := stmt.Init.(*dst.AssignStmt)
			if !ok {
				p := parentMap[stmt]
				b, ok := p.(*dst.BlockStmt)
				if !ok {
					parent = parentMap[parent]
					continue
				}
				for i, v := range b.List {
					if v == stmt {
						if i == 0 {
							break
						}
						if a, ok := b.List[i-1].(*dst.AssignStmt); ok {
							assignStmt = a
							break
						}
					}
				}
				if assignStmt == nil {
					parent = parentMap[parent]
					continue
				}
			} else {
				assignStmt = a
			}
		case *dst.BlockStmt:
			assignStmt = BlockStmt(stmt)
			if assignStmt == nil {
				parent = parentMap[parent]
				continue
			}

		}

		rhs := Rhs(assignStmt, func() { parent = parentMap[parent] })
		if rhs == nil {
			continue
		}

		return Reason(rhs), true, false
	}

	return "", false, false
}

func Reason(expr dst.Expr) string {
	switch v := expr.(type) {
	default:
		fmt.Printf("reason default: %#v\n", v)
		return ""
	case *dst.CallExpr:
		return Reason(v.Fun)
	case *dst.Ident:
		return v.Name
	case *dst.SelectorExpr:
		return Reason(v.X) + "." + v.Sel.Name
	}
}

func BlockStmt(stmt *dst.BlockStmt) *dst.AssignStmt {
	for i := len(stmt.List) - 1; i >= 0; i-- {
		assignStmt, ok := stmt.List[i].(*dst.AssignStmt)
		if !ok {
			continue
		}
		rhs := Rhs(assignStmt, nil)
		if rhs == nil {
			continue
		}
		// Здесь пропускаем errors.Join, тк странно такую причину указывать.
		// Поищем повыше
		if call, ok := rhs.(*dst.CallExpr); ok {
			if f, ok := call.Fun.(*dst.SelectorExpr); ok {
				if f.Sel.Name == "errors" {
					if ident, ok := f.X.(*dst.Ident); ok && ident.Name == "Join" {
						continue
					}
				}
			}
		}

		return assignStmt
	}

	return nil
}

func Rhs(assignStmt *dst.AssignStmt, changeParent func()) dst.Expr {
	index := -1
	for i, field := range assignStmt.Lhs {
		f, ok := field.(*dst.Ident)
		if !ok {
			continue
		}
		if strings.HasPrefix(f.Name, "err") ||
			strings.HasSuffix(f.Name, "err") ||
			strings.HasSuffix(f.Name, "Err") {
			index = i
			break
		}
	}
	if index < 0 {
		if changeParent != nil {
			changeParent()
		}

		return nil
	}
	rhs := assignStmt.Rhs[0]
	if len(assignStmt.Rhs) > 1 {
		rhs = assignStmt.Rhs[index]
	}
	return rhs
}
