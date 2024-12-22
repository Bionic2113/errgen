package example

import (
	"fmt"
	"strconv"
)

type UpdateNameError struct {
	newName string
	reason  string
	err     error
}

func NewUpdateNameError(newName string, reason string, err error) *UpdateNameError {
	return &UpdateNameError{
		newName: newName,
		reason:  reason,

		err: err,
	}
}

func (e *UpdateNameError) Error() string {
	return "[" +
		"example/" +
		"example" +
		".User" +
		"] - " +
		"UpdateName - " +
		e.reason +

		" - args: {" +

		"newName: " + e.newName +
		"}" +

		"\n" +
		e.err.Error()
}

func (e *UpdateNameError) Unwrap() error {
	return e.err
}

func (e *UpdateNameError) Is(err error) bool {
	_, ok := err.(*UpdateNameError)
	return ok
}

type ProcessUserError struct {
	user   *User
	count  int
	reason string
	err    error
}

func NewProcessUserError(user *User, count int, reason string, err error) *ProcessUserError {
	return &ProcessUserError{
		user:   user,
		count:  count,
		reason: reason,

		err: err,
	}
}

func (e *ProcessUserError) Error() string {
	return "[" +
		"example/" +
		"example" +

		"] - " +
		"ProcessUser - " +
		e.reason +

		" - args: {" +

		"user: " + fmt.Sprintf("%#v", e.user) + ", " +
		"count: " + strconv.Itoa(e.count) +
		"}" +

		"\n" +
		e.err.Error()
}

func (e *ProcessUserError) Unwrap() error {
	return e.err
}

func (e *ProcessUserError) Is(err error) bool {
	_, ok := err.(*ProcessUserError)
	return ok
}

type IsOlderError struct {
	user   *User
	count  int
	reason string
	err    error
}

func NewIsOlderError(user *User, count int, reason string, err error) *IsOlderError {
	return &IsOlderError{
		user:   user,
		count:  count,
		reason: reason,

		err: err,
	}
}

func (e *IsOlderError) Error() string {
	return "[" +
		"example/" +
		"example" +
		".User" +
		"] - " +
		"IsOlder - " +
		e.reason +

		" - args: {" +

		"user: " + fmt.Sprintf("%#v", e.user) + ", " +
		"count: " + strconv.Itoa(e.count) +
		"}" +

		"\n" +
		e.err.Error()
}

func (e *IsOlderError) Unwrap() error {
	return e.err
}

func (e *IsOlderError) Is(err error) bool {
	_, ok := err.(*IsOlderError)
	return ok
}

type IsYoungerError struct {
	user   *User
	count  int
	reason string
	err    error
}

func NewIsYoungerError(user *User, count int, reason string, err error) *IsYoungerError {
	return &IsYoungerError{
		user:   user,
		count:  count,
		reason: reason,

		err: err,
	}
}

func (e *IsYoungerError) Error() string {
	return "[" +
		"example/" +
		"example" +
		".User" +
		"] - " +
		"IsYounger - " +
		e.reason +

		" - args: {" +

		"user: " + fmt.Sprintf("%#v", e.user) + ", " +
		"count: " + strconv.Itoa(e.count) +
		"}" +

		"\n" +
		e.err.Error()
}

func (e *IsYoungerError) Unwrap() error {
	return e.err
}

func (e *IsYoungerError) Is(err error) bool {
	_, ok := err.(*IsYoungerError)
	return ok
}

type IsYoungerOrOlderError struct {
	user   *User
	count  int
	reason string
	err    error
}

func NewIsYoungerOrOlderError(user *User, count int, reason string, err error) *IsYoungerOrOlderError {
	return &IsYoungerOrOlderError{
		user:   user,
		count:  count,
		reason: reason,

		err: err,
	}
}

func (e *IsYoungerOrOlderError) Error() string {
	return "[" +
		"example/" +
		"example" +
		".User" +
		"] - " +
		"IsYoungerOrOlder - " +
		e.reason +

		" - args: {" +

		"user: " + fmt.Sprintf("%#v", e.user) + ", " +
		"count: " + strconv.Itoa(e.count) +
		"}" +

		"\n" +
		e.err.Error()
}

func (e *IsYoungerOrOlderError) Unwrap() error {
	return e.err
}

func (e *IsYoungerOrOlderError) Is(err error) bool {
	_, ok := err.(*IsYoungerOrOlderError)
	return ok
}
