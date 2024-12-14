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
	// FunctionInfo содержит метаданные о функции, которая возвращает ошибку
	// Имя пакета
	// Имя подпакета (если есть)
	// Имя функции
	// Тип получателя (для методов)

	// Аргументы функции
	// Импорты, необходимые для функции
	// Флаг наличия возвращаемой ошибки
	// ArgInfo содержит информацию об аргументе функции
	// Имя аргумента
	// Тип аргумента

	// ErrorTemplate используется для генерации файла с ошибками
	// Имя пакета
	// Список функций с ошибками
	// FileProcessor обрабатывает файлы Go и генерирует обертки для ошибок
	// newFileProcessor создает новый процессор файлов
	// processFiles обходит все файлы и обрабатывает их

	// Пропускаем директории, тесты и файлы с ошибками
	// processFile обрабатывает отдельный файл
	// generateErrorFiles генерирует файлы с обертками ошибок для каждого пакета
	// getSubPackageName возвращает имя подпакета относительно базовой директории
	// analyzeFunctions анализирует функции в файле и модифицирует их для использования оберток ошибок
	// Читаем оригинальный файл для сохранения форматирования
	// Собираем информацию о пустых строках

	// Анализируем функции
	// Записываем модифицированный файл
	// collectImports собирает информацию об импортах из файла
	// analyzeEmptyLines анализирует пустые строки в файле
	// createFunctionInfo создает информацию о функции
	// extractReceiverType извлекает тип получателя метода

	// writeModifiedFile записывает модифицированный файл с сохранением форматирования
	// Восстанавливаем форматирование
	// Skip if type couldn't be determined
	// Collect required imports

	// Analyze argument types to determine required imports
	// For structs and any
	// Handle types from other packages

	// Form imports list
	// Create structure with data for template, including imports
	// Форматируем сгенерированный код
	// Создаем конфигурацию для форматирования
	// Парсим сгенерированный код
	// Форматируем код
	// Записываем отформатированный код

	// isBasicType checks if the type is a basic Go type that doesn't need fmt
	// modifyFunctionBody analyzes and modifies function bodies to wrap error returns
	// Создаем карту отношений узлов AST
	// Получаем индекс параметра error
	// Пропускаем, если это уже обернутая ошибка
	// Определяем сообщение об ошибке и нужно ли использовать nil
	// Создаем конструктор ля обертки ошибки

	// isNilError checks if the expression is nil
	// getArgumentNames returns a list of function argument expressions
	// isErrorWrapper checks if the expression is already a wrapped error
	// Создаем мапу всех импортов
	// Проверяем использование каждого имп��рта
	// Находим соответствующий имп��рт
	// Импорт используется, удаляем из мапы

	// Проверяем использование в вызовах функций
	// Удаляем неиспользуемые импорты из декларации
	// Если все импорты удалены, помечаем декларацию для удаления
	// Удаляем пустые декларации импортов
	// extractErrorMessage извлекает сообщение об ошибке из выражения и определяет, нужно ли использовать nil для err
	// Проверяем вызовы функций (errors.New, fmt.Errorf, fmt.Error и т.д.)
	// Для errors.New и fmt.Error/Errorf используем сообщение как reason и nil как err

	// Для форматированных строк преобразуем в текст
	// findLastFunctionCall ищет последний вызо�� функции перед return в родительских узлах
type FunctionInfo struct {
	PackageName    string
	SubPackageName string
	FunctionName   string
	ReceiverType   string
	Args           []ArgInfo
	Imports  map[string]string

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
	currentDir   string
	packages     map[string][]FunctionInfo
	packagePaths map[string]string
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
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)

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

func analyzeFunctions(node *ast.File, pkgName, subPkg string, currentDir string, fileName string) []FunctionInfo {
	var functions []FunctionInfo
	imports := collectImports(node)
	originalPath := filepath.Join(currentDir, subPkg, fileName)
	originalContent, err := os.ReadFile(originalPath)
	if err != nil {
		return nil
	}
	emptyLineRuns := analyzeEmptyLines(string(originalContent))
	ast.Inspect(node, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok && hasErrorReturn(funcDecl) {
			f := createFunctionInfo(funcDecl, pkgName, subPkg, imports)
			functions = append(functions, f)
			modifyFunctionBody(funcDecl, f)

		}
		return true
	})
	if len(functions) > 0 {
		writeModifiedFile(node, originalPath, emptyLineRuns)
	}
	return functions
}
func collectImports(node *ast.File) map[string]string {
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
func analyzeEmptyLines(content string) map[int]int {
	lines := strings.Split(content, "\n")
	emptyLineRuns := make(map[int]int)
	currentRun := 0
	lastNonEmptyLine := -1

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			currentRun++
		} else {
			if currentRun > 0 && lastNonEmptyLine >= 0 {
				emptyLineRuns[lastNonEmptyLine] = 1
			}
			currentRun = 0
			lastNonEmptyLine = i
		}
	}
	if currentRun > 0 && lastNonEmptyLine >= 0 {
		emptyLineRuns[lastNonEmptyLine] = 1
	}
	return emptyLineRuns
}
func createFunctionInfo(funcDecl *ast.FuncDecl, pkgName, subPkg string, imports map[string]string) FunctionInfo {

	args := extractArgs(funcDecl)
	receiverType := extractReceiverType(funcDecl)
	return FunctionInfo{PackageName: pkgName, SubPackageName: subPkg, FunctionName: funcDecl.Name.Name, ReceiverType: receiverType, Args: args, Imports: imports, HasError: true}
}

func extractReceiverType(funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return ""
	}
	if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {

		if ident, ok := starExpr.X.(*ast.Ident); ok {
			return ident.Name
		}
	} else if ident, ok := funcDecl.Recv.List[0].Type.(*ast.Ident); ok {
		return ident.Name

	}
	return ""
}
func writeModifiedFile(node *ast.File, path string, emptyLineRuns map[int]int) {
	removeUnusedImports(node)
	var buf bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	fset := token.NewFileSet()
	if err := cfg.Fprint(&buf, fset, node); err != nil {
		return
	}

	var result []string
	lines := strings.Split(buf.String(), "\n")
	emptyLineCount := 0

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" {
			// Если это первая непустая строка после пустых строк
			if emptyLineCount > 0 {
				// Добавляем только одну пустую строку
				result = append(result, "")
				emptyLineCount = 0
			}
			result = append(result, line)
		} else {
			// Считаем последовательные пустые строки
			emptyLineCount++
			// Добавляем пустую строку только если это последняя строка файла
			if i == len(lines)-1 {
				result = append(result, "")
			}
		}
	}

	os.WriteFile(path, []byte(strings.Join(result, "\n")), 0644)
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

func getErrorReturnIndex(funcDecl *ast.FuncDecl) int {
	if funcDecl.Type.Results == nil {
		return -1
	}
	var totalIndex int
	for _, result := range funcDecl.Type.Results.List {
		if len(result.Names) == 0 {
			if ident, ok := result.Type.(*ast.Ident); ok {
				if ident.Name == "error" {
					return totalIndex
				}
			}
			totalIndex++
		} else {
			for range result.Names {
				if ident, ok := result.Type.(*ast.Ident); ok {
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

		if typeStr == "" {
			continue
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
			case arg.Type == "interface{}" || strings.Contains(arg.Type, "[]"):
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
	if err := os.WriteFile(errFilePath, buf.Bytes(), 0644); err != nil {
		panic(err)

	}
}
func isBasicType(typeName string) bool {
	basicTypes := map[string]bool{"string": true, "int": true, "int64": true, "uint64": true, "float64": true, "bool": true, "interface{}": true}
	return basicTypes[strings.TrimPrefix(typeName, "*")]
}

func modifyFunctionBody(funcDecl *ast.FuncDecl, info FunctionInfo) {
	parentMap := make(map[ast.Node]ast.Node)
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

	errorIndex := getErrorReturnIndex(funcDecl)
	if errorIndex == -1 {
		return
	}

	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		returnStmt, ok := n.(*ast.ReturnStmt)
		if !ok || errorIndex >= len(returnStmt.Results) {
			return true
		}

		result := returnStmt.Results[errorIndex]
		if isNilError(result) {
			return true
		}

		if isErrorWrapper(result) {
			return true
		}

		// Определяем сообщение об ошибке и нужно ли использовать nil
		var reason string
		var useNilError bool

		// Сначала проверяем, не является ли ошибка результатом вызова функции
		if msg, ok, useNil := findLastFunctionCall(returnStmt, parentMap); ok {
			reason = msg
			useNilError = useNil
		} else if msg, ok, useNil := extractErrorMessage(result); ok {
			// Если нет, проверяем не создается ли ошибка напрямую
			reason = msg
			useNilError = useNil
		} else {
			reason = "unknown error in " + info.FunctionName
			useNilError = false
		}

		var errArg ast.Expr
		if useNilError {
			errArg = ast.NewIdent("nil")
		} else {
			errArg = result
		}

		constructorCall := &ast.CallExpr{
			Fun: ast.NewIdent("New" + info.FunctionName + "Error"),
			Args: append(
				getArgumentNames(funcDecl),
				ast.NewIdent("\""+reason+"\""),
				errArg,
			),
		}
		returnStmt.Results[errorIndex] = constructorCall
		return true
	})
}
func isNilError(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "nil"
	}
	return false
}
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

func isErrorWrapper(expr ast.Expr) bool {
	if callExpr, ok := expr.(*ast.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*ast.Ident); ok {
			return strings.HasSuffix(ident.Name, "Error")
		}

	}
	return false
}
func removeUnusedImports(node *ast.File) {
	imports := make(map[string]*ast.ImportSpec)

	for _, imp := range node.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			imports[path] = imp
		}
	}
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:

			if ident, ok := x.X.(*ast.Ident); ok {
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
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
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
				genDecl.Specs = nil
			}
		}
	}
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
func extractErrorMessage(expr ast.Expr) (string, bool, bool) {
	switch v := expr.(type) {
	case *ast.CallExpr:
		if sel, ok := v.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if (ident.Name == "errors" && sel.Sel.Name == "New") || (ident.Name == "fmt" && (sel.Sel.Name == "Errorf" || sel.Sel.Name == "Error")) {
					if len(v.Args) > 0 {
						if lit, ok := v.Args[0].(*ast.BasicLit); ok {
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
		} else if ident, ok := v.Fun.(*ast.Ident); ok {
			return ident.Name, true, false
		}
	case *ast.Ident:
		if v.Name != "nil" {
			return v.Name, true, false
		}
	}
	return "", false, false
}
func findLastFunctionCall(node ast.Node, parentMap map[ast.Node]ast.Node) (string, bool, bool) {
	parent := parentMap[node]
	for parent != nil {
		if ifStmt, ok := parent.(*ast.IfStmt); ok {
			if assignStmt, ok := ifStmt.Init.(*ast.AssignStmt); ok {
				if len(assignStmt.Lhs) > 0 && len(assignStmt.Rhs) > 0 {
					// Проверяем, что левая часть это err
					if errIdent, ok := assignStmt.Lhs[0].(*ast.Ident); ok && errIdent.Name == "err" {
						if callExpr, ok := assignStmt.Rhs[0].(*ast.CallExpr); ok {
							// Получаем полное имя вызываемой функции
							if sel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
								if recv, ok := sel.X.(*ast.Ident); ok {
									// Для методов возвращаем receiver.method
									return recv.Name + "." + sel.Sel.Name, true, false
								}
								// Для функций пакета возвращаем pkg.func
								return sel.Sel.Name, true, false
							} else if ident, ok := callExpr.Fun.(*ast.Ident); ok {
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
