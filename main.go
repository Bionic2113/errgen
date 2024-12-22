package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

type FunctionInfo struct {
	PackageName    string
	SubPackageName string
	FunctionName   string
	ReceiverType   string
	Args           []ArgInfo
	Imports        map[string]string

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

type FileProcessor struct {
	packages     map[string][]FunctionInfo
	packagePaths map[string]string
	currentDir   string
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
	return &FileProcessor{currentDir: currentDir, packages: make(map[string][]FunctionInfo), packagePaths: make(map[string]string)}, nil
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

	pkgName := node.Name.Name
	pkgDir := filepath.Dir(path)
	subPkg := getSubPackageName(pkgDir, p.currentDir)
	fileName := filepath.Base(path)
	functions := analyzeFunctions(node, pkgName, subPkg, p.currentDir, fileName)
	if len(functions) > 0 {
		p.packages[pkgName] = append(p.packages[pkgName], functions...)
		p.packagePaths[pkgName] = pkgDir
	}
	return nil
}

func (p *FileProcessor) generateErrorFiles() {
	for pkg, functions := range p.packages {
		generateErrorFile(pkg, functions, p.packagePaths[pkg])
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

func collectImports(node *dst.File) map[string]string {
	imports := make(map[string]string)
	for _, imp := range node.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)

			name := ""
			if imp.Name != nil {
				name = imp.Name.Name
			} else {
				name = filepath.Base(path)
			}
			imports[name] = path
		}
	}
	return imports
}

func createFunctionInfo(funcDecl *dst.FuncDecl, pkgName, subPkg string, imports map[string]string) FunctionInfo {
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
		if len(result.Names) == 0 {
			if ident, ok := result.Type.(*dst.Ident); ok {
				if ident.Name == "error" {
					return totalIndex
				}
			}
			totalIndex++
		} else {
			for range result.Names {
				if ident, ok := result.Type.(*dst.Ident); ok {
					if ident.Name == "error" {
						return totalIndex
					}
				}
				totalIndex++
			}
		}
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

func generateErrorFile(pkgName string, functions []FunctionInfo, pkgPath string) {
	imports := make(map[string]string)
	for _, f := range functions {
		for _, arg := range f.Args {

			switch {
			case strings.HasPrefix(arg.Type, "time."):
				imports["time"] = "time"

			case arg.Type == "int" || arg.Type == "int64" || arg.Type == "uint64":
				imports["strconv"] = "strconv"
			case arg.Type == "float64":
				imports["strconv"] = "strconv"

			case arg.Type == "bool":
				imports["strconv"] = "strconv"
			case arg.Type == "any" || strings.Contains(arg.Type, "[]"):
				imports["fmt"] = "fmt"
			case !isBasicType(arg.Type):
				imports["fmt"] = "fmt"
			}
			if strings.Contains(arg.Type, ".") {
				parts := strings.SplitN(arg.Type, ".", 2)
				if len(parts) == 2 {
					pkgName := strings.TrimPrefix(parts[0], "*")
					if importPath, ok := f.Imports[pkgName]; ok {
						imports[pkgName] = importPath
					}
				}
			}
		}
	}

	var importsList []string
	for _, imp := range imports {
		importsList = append(importsList, fmt.Sprintf(`	"%s"`, imp))
	}
	sort.Strings(importsList)
	templateData := ErrorTemplate{Package: pkgName, Functions: functions}
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
	if err == nil {
		return nil
	}

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

func (e *{{.FunctionName}}Error) Is(err error) bool {
	_, ok := err.(*{{.FunctionName}}Error)
	return ok
} 
{{end}}`
	data := struct {
		Package   string
		Functions []FunctionInfo
		Imports   []string
	}{Package: templateData.Package, Functions: templateData.Functions, Imports: importsList}
	errFilePath := filepath.Join(pkgPath, "errors.go")
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
		} else if msg, ok, useNil := findLastFunctionCall(returnStmt, parentMap); ok {
			reason = msg
			useNilError = useNil
		} else {
			reason = "unknown error in " + info.FunctionName
			useNilError = false
		}

		var errArg dst.Expr
		if useNilError {
			errArg = dst.NewIdent("nil")
		} else {
			errArg = result
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

func isErrorWrapper(expr dst.Expr) bool {
	if callExpr, ok := expr.(*dst.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*dst.Ident); ok {
			return strings.HasSuffix(ident.Name, "Error")
		}
	}
	return false
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
						pkgName = filepath.Base(path)
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
							pkgName = filepath.Base(path)
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
					return strings.Trim(lit.Value, `"`), true, true
				}
			}
			return ident.Name, true, false
		}
	case *dst.Ident:
		if v.Name != "nil" {
			return v.Name, true, false
		}
	}
	return "", false, false
}

func findLastFunctionCall(node dst.Node, parentMap map[dst.Node]dst.Node) (string, bool, bool) {
	parent := parentMap[node]
	for parent != nil {
		if ifStmt, ok := parent.(*dst.IfStmt); ok {
			if assignStmt, ok := ifStmt.Init.(*dst.AssignStmt); ok {
				if len(assignStmt.Lhs) > 0 && len(assignStmt.Rhs) > 0 {
					// Проверяем, что левая часть это err
					if errIdent, ok := assignStmt.Lhs[0].(*dst.Ident); ok && errIdent.Name == "err" {
						if callExpr, ok := assignStmt.Rhs[0].(*dst.CallExpr); ok {
							// Получаем полное имя вызываемой функции
							if sel, ok := callExpr.Fun.(*dst.SelectorExpr); ok {
								if recv, ok := sel.X.(*dst.Ident); ok {
									// Для методов возвращаем receiver.method
									return recv.Name + "." + sel.Sel.Name, true, false
								}
								// Для функций пакета возвращаем pkg.func
								return sel.Sel.Name, true, false
							} else if ident, ok := callExpr.Fun.(*dst.Ident); ok {
								// Для локальных функций возвращаем имя функции
								return ident.Name, true, false
							}
						}
					}
				}
			}
		}
		parent = parentMap[parent]
	}
	return "", false, false
}
