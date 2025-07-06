# Error Wrapper Generator

A tool for generating error wrappers in Go that provides detailed error context including function name, arguments, and call chain.

## Features

- Automatically generates error wrapper types for functions that return errors
- Wraps errors with context about where and why they occurred
- Preserves error chains with `Unwrap(), Is() and As()` support
- Includes function arguments in error messages
- Handles methods on structs and package-level functions
- Supports all basic Go types and custom types
- Automatically modifies existing error returns to use wrappers

## Installation

```bash
go install github.com/Bionic2113/errgen@latest
```

## Usage

Run in your project directory:

```bash
errgen
```

This will:
1. Scan all .go files in the current directory and subdirectories
2. Generate error wrapper types for functions that return errors
3. Create `errors.go` files in each package containing the wrappers
4. Modify existing error returns to use the new wrappers
5. It can skip function arguments using the `pkg/skipper` package
6. It can skip structure fields, using the `pkg/stringer` package

## Example

See more [in example folder](./example)

Config:

```yaml
wrapper_filename: "errwrap_gen"
simple_err_filename: "error_gen"
skipper:
  skip_types:
    "github.com/Bionic2113/errgen/pkg/skipper":
      all: false
      names:
        - "Config"
  with_default: true
  rules:
    - type: suffix
      value: "skip.go"
stringer:
  separator: "\\n"
  connector: ": "
  filename: "strings"
  tagname: "errgen"
```

Given this code:

```go
func ProcessUser(user *User, count int) error {
	if err := user.UpdateName("New"); err != nil {
		return err
	}

	return errors.New("processing failed")
}
```

The generator will create a wrapper and modify the code to:

```go
var ErrExample2 = errors.New("processing failed")

func ProcessUser(user *User, count int) error {
	if err := user.UpdateName("New"); err != nil {
		return NewProcessUserError(user, count, "user.UpdateName", err)
	}

	return NewProcessUserError(user, count, "unknown error in ProcessUser", ErrExample2)
}

type ProcessUserError struct {
	user         *User
	count        int
	reasonErrGen string
	errErrGen    error
}

func NewProcessUserError(user *User, count int, reasonErrGen string, errErrGen error) *ProcessUserError {
	return &ProcessUserError{
		user:         user,
		count:        count,
		reasonErrGen: reasonErrGen,
		errErrGen:    errErrGen,
	}
}

func (e *ProcessUserError) Error() string {
	return "[" + "example" + "] - " +
		"ProcessUser - " + e.reasonErrGen +
		" - args: {" +
		"user: " + fmt.Sprintf("%#v", e.user) + ", " +
		"count: " + strconv.Itoa(e.count) +
		"}" + "\n" +
		e.errErrGen.Error()
}

func (e *ProcessUserError) Unwrap() error {
	return e.errErrGen
}

func (e *ProcessUserError) Is(target error) bool {
	if _, ok := target.(*ProcessUserError); ok {
		return true
	}
	return errors.Is(e.errErrGen, target)
}

```

This provides rich error context while maintaining the original error chain.
