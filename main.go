package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

var versionMatch *regexp.Regexp = regexp.MustCompile(`^(/?\w+(\.?|-?))+(/v\d+)$`)

// (`^(/?\w+(\.?|-?\w+))+(/v\d+)$`) //(`^\w+/\w+(/v\d+)$`)

var prefixAndSuffixMatch *regexp.Regexp = regexp.MustCompile(`(go-\w+)|(\w+-go)`)

//(`^(\w+(\.?|-?\w+)*/)+((go-\w+)|(\w+\-go))$`)

type FunctionInfo struct {
	PackageName    string
	SubPackageName string
	FunctionName   string
	ReceiverType   string
	Args           []ArgInfo
	Imports        map[string]Path

	HasError bool
}

type ArgInfo struct {
	Name string
	Type string
}

type ErrorTemplate struct {
	Package   string
	Functions []FunctionInfo
}

type PkgInfo struct {
	Name string
	Path Path
}

type Path struct {
	Alias string
	Path  string
}

type FileProcessor struct {
	packages   map[PkgInfo][]FunctionInfo
	currentDir string
}

func main() {
	processor, err := newFileProcessor()
	if err != nil {
		panic(err)
	}
	if err := processor.processFiles(); err != nil {
		panic(err)
	}
	processor.generateErrorFiles()
}

func newFileProcessor() (*FileProcessor, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return &FileProcessor{currentDir: currentDir, packages: make(map[PkgInfo][]FunctionInfo)}, nil
}

func (p *FileProcessor) processFiles() error {
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
			strings.HasSuffix(path, "main.go") {
			return nil
		}

		return p.processFile(path)
	})
}

func (p *FileProcessor) processFile(path string) error {
	fset := token.NewFileSet()
	node, err := decorator.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	pkgInfo := PkgInfo{Name: node.Name.Name, Path: Path{Path: filepath.Dir(path)}}
	// пропустим моки
	if strings.HasSuffix(pkgInfo.Path.Path, "mocks") {
		return nil
	}
	subPkg := getSubPackageName(pkgInfo.Path.Path, p.currentDir)
	fileName := filepath.Base(path)
	functions := analyzeFunctions(node, pkgInfo.Name, subPkg, p.currentDir, fileName)
	if len(functions) > 0 {
		p.packages[pkgInfo] = append(p.packages[pkgInfo], functions...)
	}
	return nil
}

func (p *FileProcessor) generateErrorFiles() {
	for pkg, functions := range p.packages {
		generateErrorFile(pkg, functions)
	}
}

func getSubPackageName(pkgDir, baseDir string) string {
	rel, err := filepath.Rel(baseDir, pkgDir)
	if err != nil {
		return ""
	}
	if rel == "." {
		return ""
	}
	return rel
}

func analyzeFunctions(node *dst.File, pkgName, subPkg string, currentDir string, fileName string) []FunctionInfo {
	var functions []FunctionInfo
	imports := collectImports(node)
	originalPath := filepath.Join(currentDir, subPkg, fileName)

	dst.Inspect(node, func(n dst.Node) bool {
		if funcDecl, ok := n.(*dst.FuncDecl); ok && hasErrorReturn(funcDecl) {
			f := createFunctionInfo(funcDecl, pkgName, subPkg, imports)
			functions = append(functions, f)
			modifyFunctionBody(funcDecl, f)

		}
		return true
	})
	if len(functions) > 0 {
		writeModifiedFile(node, originalPath)
	}
	return functions
}

func collectImports(node *dst.File) map[string]Path {
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

		imports[nameFromPath(path)] = Path{Path: path}
	}
	return imports
}

func nameFromPath(path string) string {
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

func createFunctionInfo(funcDecl *dst.FuncDecl, pkgName, subPkg string, imports map[string]Path) FunctionInfo {
	args := extractArgs(funcDecl)
	receiverType := extractReceiverType(funcDecl)
	return FunctionInfo{PackageName: pkgName, SubPackageName: subPkg, FunctionName: funcDecl.Name.Name, ReceiverType: receiverType, Args: args, Imports: imports, HasError: true}
}

func extractReceiverType(funcDecl *dst.FuncDecl) string {
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

func writeModifiedFile(node *dst.File, path string) {
	removeUnusedImports(node)

	var buf bytes.Buffer
	if err := decorator.Fprint(&buf, node); err != nil {
		fmt.Printf("Error formatting modified file: %v\n", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		fmt.Printf("Error writing modified file: %v\n", err)
	}
}

func hasErrorReturn(funcDecl *dst.FuncDecl) bool {
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

func getErrorReturnIndex(funcDecl *dst.FuncDecl) int {
	if funcDecl.Type.Results == nil {
		return -1
	}
	var totalIndex int
	for _, result := range funcDecl.Type.Results.List {
		// if len(result.Names) == 0 {
		if ident, ok := result.Type.(*dst.Ident); ok {
			if ident.Name == "error" {
				return totalIndex
			}
		}
		totalIndex++
		// } else {
		// 	for range result.Names {
		// 		if ident, ok := result.Type.(*dst.Ident); ok {
		// 			if ident.Name == "error" {
		// 				return totalIndex
		// 			}
		// 		}
		// 		totalIndex++
		// 	}
		// }
	}
	return -1
}

func extractArgs(funcDecl *dst.FuncDecl) []ArgInfo {
	var args []ArgInfo
	if funcDecl.Type.Params == nil {
		return args
	}
	for _, field := range funcDecl.Type.Params.List {
		typeStr := ""
		switch t := field.Type.(type) {
		default:
			typeStr = "any"
		case *dst.Ident:
			typeStr = t.Name
		case *dst.StarExpr:
			if ident, ok := t.X.(*dst.Ident); ok {
				typeStr = "*" + ident.Name
			} else if sel, ok := t.X.(*dst.SelectorExpr); ok {
				if ident, ok := sel.X.(*dst.Ident); ok {
					typeStr = "*" + ident.Name + "." + sel.Sel.Name
				}
			}
		case *dst.ArrayType:
			if ident, ok := t.Elt.(*dst.Ident); ok {
				typeStr = "[]" + ident.Name
			} else if sel, ok := t.Elt.(*dst.SelectorExpr); ok {
				if ident, ok := sel.X.(*dst.Ident); ok {
					typeStr = "[]" + ident.Name + "." + sel.Sel.Name
				}
			} else if t, ok := t.Elt.(*dst.StarExpr); ok {
				if ident, ok := t.X.(*dst.Ident); ok {
					typeStr = "[]*" + ident.Name
				} else if sel, ok := t.X.(*dst.SelectorExpr); ok {
					if ident, ok := sel.X.(*dst.Ident); ok {
						typeStr = "[]*" + ident.Name + "." + sel.Sel.Name
					}
				}
			}
		case *dst.SelectorExpr:
			if ident, ok := t.X.(*dst.Ident); ok {
				typeStr = ident.Name + "." + t.Sel.Name
			}
		case *dst.InterfaceType:
			typeStr = "any"
		}

		for _, name := range field.Names {
			args = append(args, ArgInfo{Name: name.Name, Type: typeStr})
		}
	}
	return args
}

func generateErrorFile(pkgInfo PkgInfo, functions []FunctionInfo) {
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
			case !isBasicType(arg.Type):
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

	var importsList []string
	for _, imp := range imports {
		importsList = append(importsList, imp.Alias+" "+`"`+imp.Path+`"`)
	}
	sort.Strings(importsList)
	templateData := ErrorTemplate{Package: pkgInfo.Name, Functions: functions}
	tmpl := `package {{.Package}}
import (
{{range .Imports}}{{.}}
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
		Imports   []string
	}{Package: templateData.Package, Functions: templateData.Functions, Imports: importsList}
	errFilePath := filepath.Join(pkgInfo.Path.Path, "errors.go")
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

func isBasicType(typeName string) bool {
	basicTypes := map[string]bool{"string": true, "int": true, "int64": true, "uint64": true, "float64": true, "bool": true, "any": true}
	return basicTypes[strings.TrimPrefix(typeName, "*")]
}

func modifyFunctionBody(funcDecl *dst.FuncDecl, info FunctionInfo) {
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

	errorIndex := getErrorReturnIndex(funcDecl)
	if errorIndex == -1 {
		return
	}

	dst.Inspect(funcDecl.Body, func(n dst.Node) bool {
		returnStmt, ok := n.(*dst.ReturnStmt)
		if !ok || errorIndex >= len(returnStmt.Results) {
			return true
		}

		result := returnStmt.Results[errorIndex]
		if isNilError(result) {
			return true
		}

		if !isNeedChange(result) {
			return true
		}

		// if isErrorWrapper(result) {
		// 	return true
		// }

		// Определяем сообщение об ошибке и нужно ли использовать nil
		var reason string
		var useNilError bool

		// Проверяем не создается ли ошибка напрямую
		if msg, ok, useNil := extractErrorMessage(result); ok {
			reason = msg
			useNilError = useNil
			// Проверяем, не является ли ошибка результатом вызова функции
		} else if msg, ok, funcLit := findLastFunctionCall(returnStmt, parentMap); ok || funcLit {
			if funcLit {
				return true
			}
			reason = msg
		} else {
			reason = "unknown error in " + info.FunctionName
		}

		errArg := isErrorWrapper(result)
		if errArg == nil {
			if useNilError {
				errArg = dst.NewIdent("nil")
			} else {
				errArg = result
			}
		}

		constructorCall := &dst.CallExpr{
			Fun: dst.NewIdent("New" + info.FunctionName + "Error"),
			Args: append(
				getArgumentNames(funcDecl),
				dst.NewIdent("\""+reason+"\""),
				errArg,
			),
		}
		returnStmt.Results[errorIndex] = constructorCall
		return true
	})
}

func isNilError(expr dst.Expr) bool {
	if ident, ok := expr.(*dst.Ident); ok {
		return ident.Name == "nil"
	}
	return false
}

// Проверяем нужно ли нам заменить возврат ошибки
// на обертку
func isNeedChange(expr dst.Expr) bool {
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

func getArgumentNames(funcDecl *dst.FuncDecl) []dst.Expr {
	var args []dst.Expr
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			for _, name := range field.Names {
				args = append(args, dst.NewIdent(name.Name))
			}
		}
	}
	return args
}

func isErrorWrapper(expr dst.Expr) dst.Expr {
	if callExpr, ok := expr.(*dst.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*dst.Ident); ok {
			if strings.HasSuffix(ident.Name, "Error") {
				return callExpr.Args[len(callExpr.Args)-1]
			}
		}
	}
	return nil
}

func removeUnusedImports(node *dst.File) {
	imports := make(map[string]*dst.ImportSpec)

	for _, imp := range node.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			imports[path] = imp
		}
	}
	dst.Inspect(node, func(n dst.Node) bool {
		switch x := n.(type) {
		case *dst.SelectorExpr:

			if ident, ok := x.X.(*dst.Ident); ok {
				for path, imp := range imports {
					pkgName := ""
					if imp.Name != nil {
						pkgName = imp.Name.Name
					} else {
						pkgName = nameFromPath(path)
					}
					if ident.Name == pkgName {
						delete(imports, path)
					}
				}
			}
		case *dst.CallExpr:
			if sel, ok := x.Fun.(*dst.SelectorExpr); ok {
				if ident, ok := sel.X.(*dst.Ident); ok {
					for path, imp := range imports {
						pkgName := ""
						if imp.Name != nil {
							pkgName = imp.Name.Name
						} else {
							pkgName = nameFromPath(path)
						}
						if ident.Name == pkgName {
							delete(imports, path)
						}
					}
				}
			}
		}
		return true
	})

	var newImports []dst.Spec
	for _, imp := range node.Decls {
		if genDecl, ok := imp.(*dst.GenDecl); ok && genDecl.Tok == token.IMPORT {
			for _, spec := range genDecl.Specs {
				if importSpec, ok := spec.(*dst.ImportSpec); ok {
					path := strings.Trim(importSpec.Path.Value, `"`)
					if _, unused := imports[path]; !unused {
						newImports = append(newImports, importSpec)
					}
				}
			}
			if len(newImports) > 0 {
				genDecl.Specs = newImports
			} else {
				genDecl.Specs = nil
			}
		}
	}
	var newDecls []dst.Decl
	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok && genDecl.Tok == token.IMPORT {
			if len(genDecl.Specs) > 0 {
				newDecls = append(newDecls, decl)
			}
		} else {
			newDecls = append(newDecls, decl)
		}
	}
	node.Decls = newDecls
}

func extractErrorMessage(expr dst.Expr) (string, bool, bool) {
	switch v := expr.(type) {
	case *dst.CallExpr:
		if sel, ok := v.Fun.(*dst.SelectorExpr); ok {
			if ident, ok := sel.X.(*dst.Ident); ok {
				if (ident.Name == "errors" && sel.Sel.Name == "New") || (ident.Name == "fmt" && (sel.Sel.Name == "Errorf")) {
					if len(v.Args) > 0 {
						if lit, ok := v.Args[0].(*dst.BasicLit); ok {
							return strings.Trim(lit.Value, `"`), true, true
						}

						var buf bytes.Buffer
						printer.Fprint(&buf, token.NewFileSet(), v.Args[0])
						return buf.String(), true, true
					}
				}
				return ident.Name + "." + sel.Sel.Name, true, false
			}
			return sel.Sel.Name, true, false
		} else if ident, ok := v.Fun.(*dst.Ident); ok {
			// Если уже была обертка, то забираем причину
			if strings.HasSuffix(ident.Name, "Error") && len(v.Args) > 1 {
				if lit, ok := v.Args[len(v.Args)-2].(*dst.BasicLit); ok {
					return strings.Trim(lit.Value, `"`), true, false
				}
			}
			return ident.Name, true, false
		}
	}
	return "", false, false
}

func findLastFunctionCall(node dst.Node, parentMap map[dst.Node]dst.Node) (string, bool, bool) {
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
		case *dst.IfStmt:
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
			assignStmt = blockStmt(stmt)
			if assignStmt == nil {
				parent = parentMap[parent]
				continue
			}

		}

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
			parent = parentMap[parent]
			continue
		}
		rhs := assignStmt.Rhs[0]
		if len(assignStmt.Rhs) > 1 {
			rhs = assignStmt.Rhs[index]
		}

		return reason(rhs), true, false
	}
	return "", false, false
}

func reason(expr dst.Expr) string {
	switch v := expr.(type) {
	default:
		fmt.Printf("reason default: %#v\n", v)
		return ""
	case *dst.CallExpr:
		return reason(v.Fun)
	case *dst.Ident:
		return v.Name
	case *dst.SelectorExpr:
		return reason(v.X) + "." + v.Sel.Name
	}
}

func blockStmt(stmt *dst.BlockStmt) *dst.AssignStmt {
	for i := len(stmt.List) - 1; i >= 0; i-- {
		assignStmt, ok := stmt.List[i].(*dst.AssignStmt)
		if !ok {
			continue
		}
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
			continue
		}
		rhs := assignStmt.Rhs[0]
		if len(assignStmt.Rhs) > 1 {
			rhs = assignStmt.Rhs[index]
		}
		// Да, всё это выглядит убого. Потом перепишу.
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
