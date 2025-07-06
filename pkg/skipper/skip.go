package skipper

import (
	"bufio"
	"log/slog"
	"os"
	"slices"
	"strings"
)

func (s *Skipper) NeedSkipField(name, path string) bool {
	info, ok := s.SkipTypes[path]
	if !ok {
		return false
	}

	if info.All {
		return true
	}

	return slices.Contains(info.Names, name)
}

func (s *Skipper) NeedSkipFile(path string) bool {
	var skip bool
	for _, rule := range s.Rules {
		switch rule.Type {
		default:
			skip = strings.Contains(path, rule.Value)
		case prefix:
			skip = strings.HasPrefix(path, rule.Value)
		case suffix:
			skip = strings.HasSuffix(path, rule.Value)
		case dir, directory:
			skip = strings.Contains(path, "/"+rule.Value+"/")
		case contains:
			skip = strings.Contains(path, rule.Value)
		}

		if skip {
			return true
		}
	}

	return false
}

func (s *Skipper) ModuleName(path string) string {
	return s.module + strings.TrimPrefix(path, s.workDir)
}

func (s *Skipper) workDirAndModule() {
	wd, err := os.Getwd()
	if err != nil {
		s.l.Error("os.Getwd", slog.String("error", err.Error()))
		return
	}
	s.l.Info("Work directory is loaded", slog.String("WorkDir", wd))

	s.workDir = wd

	file, err := os.Open(wd + "/go.mod")
	if err != nil {
		s.l.Error("os.Open go.mod", slog.String("error", err.Error()))
		return
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

	s.module = module

	err = scanner.Err()
	if err != nil {
		s.l.Error("scanner.Err", slog.String("error", err.Error()))
	}
}
