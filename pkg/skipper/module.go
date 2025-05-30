package skipper

import (
	"bufio"
	"os"
	"strings"
)

// TODO: logger for logs
func workDirAndModule() (string, string) {
	wd, err := os.Getwd()
	if err != nil {
		println("os.Getwd error: ", err.Error())
		return "", ""
	}
	println("WorkDir: ", wd)

	file, err := os.Open(wd + "/go.mod")
	if err != nil {
		println("os.Open error: ", err.Error())
		return wd, ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var module string
	prefix := "module "

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			module = line[len(prefix):]
			break
		}
	}

	err = scanner.Err()
	if err != nil {
		println("scanner.Err error: ", err.Error())
		return wd, module
	}

	return wd, module
}

func ModuleName(path string) string {
	return module + strings.TrimPrefix(path, workDir)
}
