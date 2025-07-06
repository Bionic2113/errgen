package example

import "errors"

type Skip struct {
	Name string
	Age  int
}

func SkipCheck(name string) error {
	if name == "" {
		return errors.New("empty name")
	}

	return nil
}
