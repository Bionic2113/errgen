package example

import "errors"

var (
	ErrExample4 = errors.New("current user is nil")
	ErrExample1 = errors.New("name cannot be empty")
	ErrExample2 = errors.New("processing failed")
	ErrExample3 = errors.New("user is nil")
)
