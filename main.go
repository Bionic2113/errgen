package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// FunctionInfo contains metadata about a function that returns an error
type FunctionInfo struct {
	PackageName    string
	SubPackageName string
	FunctionName   string
	ReceiverType   string
	Args           []ArgInfo
	Imports        map[string]string
	HasError       bool
}

// ArgInfo contains information about a function argument
type ArgInfo struct {
	Name string
	Type string
}

// ErrorTemplate is used for error file generation
type ErrorTemplate struct {
	Package   string
	Functions []FunctionInfo
}

func main() {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	packages := make(map[string][]FunctionInfo)
	packagePaths := make(map[string]string)

	err = filepath.Walk(currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		if strings.HasSuffix(path, "errors.go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		pkgName := node.Name.Name
		pkgDir := filepath.Dir(path)
		subPkg := getSubPackageName(pkgDir, currentDir)
		fileName := filepath.Base(path)

		functions := analyzeFunctions(node, pkgName, subPkg, currentDir, fileName)
		if len(functions) > 0 {
			packages[pkgName] = append(packages[pkgName], functions...)
			packagePaths[pkgName] = pkgDir
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	for pkg, functions := range packages {
		generateErrorFile(pkg, functions, packagePaths[pkg])
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

func analyzeFunctions(node *ast.File, pkgName, subPkg string, currentDir string, fileName string) []FunctionInfo {
	var functions []FunctionInfo

	// Collect imports from the file
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

	ast.Inspect(node, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if hasErrorReturn(funcDecl) {
				args := extractArgs(funcDecl)

				receiverType := ""
				if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
					if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
						if ident, ok := starExpr.X.(*ast.Ident); ok {
							receiverType = ident.Name
						}
					} else if ident, ok := funcDecl.Recv.List[0].Type.(*ast.Ident); ok {
						receiverType = ident.Name
					}
				}

				f := FunctionInfo{
					PackageName:    pkgName,
					SubPackageName: subPkg,
					FunctionName:   funcDecl.Name.Name,
					ReceiverType:   receiverType,
					Args:           args,
					Imports:        imports,
					HasError:       true,
				}
				functions = append(functions, f)

				// Modify function body
				modifyFunctionBody(funcDecl, f)
			}
		}
		return true
	})

	// Write modified file
	if len(functions) > 0 {
		removeUnusedImports(node)

		fset := token.NewFileSet()
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, node); err != nil {
			return functions
		}

		// Use original file path
		originalPath := filepath.Join(
			currentDir,
			subPkg,
			fileName,
		)

		// Write changes back to file
		if err := os.WriteFile(originalPath, buf.Bytes(), 0644); err != nil {
			return functions
		}
	}

	return functions
}

func hasErrorReturn(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Results == nil {
		return false
	}

	for _, result := range funcDecl.Type.Results.List {
		if ident, ok := result.Type.(*ast.Ident); ok {
			if ident.Name == "error" {
				return true
			}
		}
	}
	return false
}

func extractArgs(funcDecl *ast.FuncDecl) []ArgInfo {
	var args []ArgInfo

	if funcDecl.Type.Params == nil {
		return args
	}

	for _, field := range funcDecl.Type.Params.List {
		typeStr := ""
		switch t := field.Type.(type) {
		case *ast.Ident:
			typeStr = t.Name
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				typeStr = "*" + ident.Name
			} else if sel, ok := t.X.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					typeStr = "*" + ident.Name + "." + sel.Sel.Name
				}
			}
		case *ast.ArrayType:
			if ident, ok := t.Elt.(*ast.Ident); ok {
				typeStr = "[]" + ident.Name
			} else if sel, ok := t.Elt.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					typeStr = "[]" + ident.Name + "." + sel.Sel.Name
				}
			}
		case *ast.SelectorExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				typeStr = ident.Name + "." + t.Sel.Name
			}
		case *ast.InterfaceType:
			typeStr = "interface{}"
		}

		// Skip if type couldn't be determined
		if typeStr == "" {
			continue
		}

		for _, name := range field.Names {
			args = append(args, ArgInfo{
				Name: name.Name,
				Type: typeStr,
			})
		}
	}

	return args
}

func generateErrorFile(pkgName string, functions []FunctionInfo, pkgPath string) {
	// Collect required imports
	imports := make(map[string]string)

	// Analyze argument types to determine required imports
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
			case arg.Type == "interface{}" || strings.Contains(arg.Type, "[]"):
				imports["fmt"] = "fmt"
			case !isBasicType(arg.Type): // For structs and any
				imports["fmt"] = "fmt"
			}

			// Handle types from other packages
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

	// Form imports list
	var importsList []string
	for _, imp := range imports {
		importsList = append(importsList, fmt.Sprintf(`	"%s"`, imp))
	}
	sort.Strings(importsList)

	templateData := ErrorTemplate{
		Package:   pkgName,
		Functions: functions,
	}

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
{{end}}`

	// Create structure with data for template, including imports
	data := struct {
		Package   string
		Functions []FunctionInfo
		Imports   []string
	}{
		Package:   templateData.Package,
		Functions: templateData.Functions,
		Imports:   importsList,
	}

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

	err = t.Execute(f, data)
	if err != nil {
		panic(err)
	}

}

// isBasicType checks if the type is a basic Go type that doesn't need fmt
func isBasicType(typeName string) bool {
	basicTypes := map[string]bool{
		"string":      true,
		"int":         true,
		"int64":       true,
		"uint64":      true,
		"float64":     true,
		"bool":        true,
		"interface{}": true,
	}
	return basicTypes[strings.TrimPrefix(typeName, "*")]
}

// modifyFunctionBody analyzes and modifies function bodies to wrap error returns
func modifyFunctionBody(funcDecl *ast.FuncDecl, info FunctionInfo) {
	// Create a map to store AST node relationships
	parentMap := make(map[ast.Node]ast.Node)

	// Fill the parent map
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		ast.Inspect(n, func(child ast.Node) bool {
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

	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		if returnStmt, ok := n.(*ast.ReturnStmt); ok {
			for i, result := range returnStmt.Results {
				if !isNilError(result) {
					// Get the name of the called function for the reason
					var calledFunc string
					var errArg ast.Expr
					// Check if the result is an error identifier
					if ident, ok := result.(*ast.Ident); ok {
						// Look for the last function call before return
						parent := parentMap[returnStmt]
						for parent != nil {
							if ifStmt, ok := parent.(*ast.IfStmt); ok {
								if assignStmt, ok := ifStmt.Init.(*ast.AssignStmt); ok {
									// Check if the returned variable matches the one in if
									if len(assignStmt.Lhs) > 0 {
										if errIdent, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
											if errIdent.Name == ident.Name {
												if callExpr, ok := assignStmt.Rhs[0].(*ast.CallExpr); ok {
													// Handle struct method call
													if selector, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
														if recv, ok := selector.X.(*ast.Ident); ok {
															calledFunc = recv.Name + "." + selector.Sel.Name
														} else {
															calledFunc = selector.Sel.Name
														}
													} else if ident, ok := callExpr.Fun.(*ast.Ident); ok {
														calledFunc = ident.Name
													}
													break
												}
											}
										}
									}
								}
							}
							parent = parentMap[parent]
						}
					} else if callExpr, ok := result.(*ast.CallExpr); ok {
						// Direct function call in return
						if selector, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
							if recv, ok := selector.X.(*ast.Ident); ok {
								if recv.Name == "errors" && selector.Sel.Name == "New" {
									// Для errors.New используем сообщение как reason, а err = nil
									calledFunc = string(callExpr.Args[0].(*ast.BasicLit).Value)
									if strings.HasPrefix(calledFunc, "\"") && strings.HasSuffix(calledFunc, "\"") {
										calledFunc = calledFunc[1 : len(calledFunc)-1]
									}
									errArg = ast.NewIdent("nil")
									goto createConstructor
								}
								calledFunc = recv.Name + "." + selector.Sel.Name
							} else {
								calledFunc = selector.Sel.Name
							}
						} else if ident, ok := callExpr.Fun.(*ast.Ident); ok {
							calledFunc = ident.Name
						}
					}

					// Skip if it's already a wrapped error
					if isErrorWrapper(result) {
						continue
					}

					// Use default reason if function name couldn't be determined
					if calledFunc == "" {
						calledFunc = "TODO: fill me up" + info.FunctionName
					}
					errArg = result

				createConstructor:
					// Create constructor call for the error wrapper
					constructorCall := &ast.CallExpr{
						Fun: ast.NewIdent("New" + info.FunctionName + "Error"),
						Args: append(
							getArgumentNames(funcDecl),
							ast.NewIdent("\""+calledFunc+"\""),
							errArg,
						),
					}
					returnStmt.Results[i] = constructorCall
				}
			}
		}
		return true
	})
}

// isNilError checks if the expression is nil
func isNilError(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "nil"
	}
	return false
}

// getArgumentNames returns a list of function argument expressions
func getArgumentNames(funcDecl *ast.FuncDecl) []ast.Expr {
	var args []ast.Expr
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			for _, name := range field.Names {
				args = append(args, ast.NewIdent(name.Name))
			}
		}
	}
	return args
}

// isErrorWrapper checks if the expression is already a wrapped error
func isErrorWrapper(expr ast.Expr) bool {
	if callExpr, ok := expr.(*ast.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*ast.Ident); ok {
			return strings.HasSuffix(ident.Name, "Error")
		}
	}
	return false
}

func removeUnusedImports(node *ast.File) {
	// Создаем мапу всех импортов
	imports := make(map[string]*ast.ImportSpec)
	for _, imp := range node.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			imports[path] = imp
		}
	}

	// Проверяем использование каждого импорта
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if ident, ok := x.X.(*ast.Ident); ok {
				// Находим соответствующий импорт
				for path, imp := range imports {
					pkgName := ""
					if imp.Name != nil {
						pkgName = imp.Name.Name
					} else {
						pkgName = filepath.Base(path)
					}
					if ident.Name == pkgName {
						delete(imports, path) // Импорт используется, удаляем из мапы
					}
				}
			}
		}
		return true
	})

	// Удаляем неиспользуемые импорты из декларации
	var newImports []ast.Spec
	for _, imp := range node.Decls {
		if genDecl, ok := imp.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			for _, spec := range genDecl.Specs {
				if importSpec, ok := spec.(*ast.ImportSpec); ok {
					path := strings.Trim(importSpec.Path.Value, `"`)
					if _, unused := imports[path]; !unused {
						newImports = append(newImports, importSpec)
					}
				}
			}
			if len(newImports) > 0 {
				genDecl.Specs = newImports
			} else {
				// Если все импорты удалены, помечаем декларацию для удаления
				genDecl.Specs = nil
			}
		}
	}

	// Удаляем пустые декларации импортов
	var newDecls []ast.Decl
	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			if len(genDecl.Specs) > 0 {
				newDecls = append(newDecls, decl)
			}
		} else {
			newDecls = append(newDecls, decl)
		}
	}
	node.Decls = newDecls
}
