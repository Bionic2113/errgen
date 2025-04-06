package utils

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
	Path string
}

type Path struct {
	Alias string
	Path  string
}
