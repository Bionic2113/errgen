package skipper

var defaultSkipTypes = map[string]pkgInfo{
	"sync":         {All: true},
	"context":      {All: true},
	"database/sql": {Names: []string{"DB", "Conn", "Tx"}},
}

var defaultRules = []Rule{
	{Type: suffix, Value: "_test.go"},
	{Type: suffix, Value: "_mock.go"},
	{Type: suffix, Value: ".pg.go"},
	{Type: suffix, Value: "main.go"},
	{Type: dir, Value: "vendor"},
	{Type: dir, Value: "mock"},
	{Type: dir, Value: "mocks"},
}

type pkgInfo struct {
	All   bool     `yaml:"all"`
	Names []string `yaml:"names"`
}

type RuleType string

const (
	prefix    RuleType = "prefix"
	suffix    RuleType = "suffix"
	directory RuleType = "directory"
	dir       RuleType = "dir"
	contains  RuleType = "contains"
)

type Rule struct {
	Type  RuleType `yaml:"type"`
	Value string   `yaml:"value"`
}
