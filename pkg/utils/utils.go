package utils

import (
	"bytes"
	"fmt"
	"go/token"
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

func IsBasicType(typeName string) bool {
	basicTypes := map[string]bool{
		"int":     true,
		"int8":    true,
		"int16":   true,
		"int32":   true,
		"int64":   true,
		"uint":    true,
		"uint8":   true,
		"uint16":  true,
		"uint32":  true,
		"uint64":  true,
		"uintptr": true,

		"float32": true,
		"float64": true,

		"complex64":  true,
		"complex128": true,

		"bool": true,

		"string": true,
		"byte":   true,
		"rune":   true,

		"any":   true,
		"error": true,
	}
	return basicTypes[strings.TrimPrefix(typeName, "*")]
}

func Convert(typeName string) string {
	switch typeName {
	default:
		return "%#v"
	case "bool":
		return "%t"
	case "string":
		return "%s"
	case "int", "int8", "int16", "int32", "int64":
		return "%d"
	case "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return "%d"
	case "float32", "float64":
		return "%f"
	case "complex64", "complex128":
		return "%v"
	case "byte":
		return "%c"
	case "rune":
		return "%c"
	}
}

func ArgumentNames(funcDecl *dst.FuncDecl, args []ArgInfo) []dst.Expr {
	result := make([]dst.Expr, len(args))
	for i, v := range args {
		result[i] = dst.NewIdent(v.Name)
	}

	return result
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
