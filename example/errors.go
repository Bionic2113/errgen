package example

import (
	"errors"
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
		err:     err,
	}
}

func (e *UpdateNameError) Error() string {
	return "[" +

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

func (e *UpdateNameError) Is(target error) bool {
	if _, ok := target.(*UpdateNameError); ok {
		return true
	}
	return errors.Is(e.err, target)
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
		err:    err,
	}
}

func (e *ProcessUserError) Error() string {
	return "[" +

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

func (e *ProcessUserError) Is(target error) bool {
	if _, ok := target.(*ProcessUserError); ok {
		return true
	}
	return errors.Is(e.err, target)
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
		err:    err,
	}
}

func (e *IsOlderError) Error() string {
	return "[" +

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

func (e *IsOlderError) Is(target error) bool {
	if _, ok := target.(*IsOlderError); ok {
		return true
	}
	return errors.Is(e.err, target)
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
		err:    err,
	}
}

func (e *IsYoungerError) Error() string {
	return "[" +

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

func (e *IsYoungerError) Is(target error) bool {
	if _, ok := target.(*IsYoungerError); ok {
		return true
	}
	return errors.Is(e.err, target)
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
		err:    err,
	}
}

func (e *IsYoungerOrOlderError) Error() string {
	return "[" +

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

func (e *IsYoungerOrOlderError) Is(target error) bool {
	if _, ok := target.(*IsYoungerOrOlderError); ok {
		return true
	}
	return errors.Is(e.err, target)
}

type FindNameError struct {
	name   string
	reason string
	err    error
}

func NewFindNameError(name string, reason string, err error) *FindNameError {
	return &FindNameError{
		name:   name,
		reason: reason,
		err:    err,
	}
}

func (e *FindNameError) Error() string {
	return "[" +

		"example" +
		".User" +
		"] - " +
		"FindName - " +
		e.reason +

		" - args: {" +

		"name: " + e.name +
		"}" +

		"\n" +
		e.err.Error()
}

func (e *FindNameError) Unwrap() error {
	return e.err
}

func (e *FindNameError) Is(target error) bool {
	if _, ok := target.(*FindNameError); ok {
		return true
	}
	return errors.Is(e.err, target)
}

type LockError struct {
	name   string
	reason string
	err    error
}

func NewLockError(name string, reason string, err error) *LockError {
	return &LockError{
		name:   name,
		reason: reason,
		err:    err,
	}
}

func (e *LockError) Error() string {
	return "[" +

		"example" +
		".User" +
		"] - " +
		"Lock - " +
		e.reason +

		" - args: {" +

		"name: " + e.name +
		"}" +

		"\n" +
		e.err.Error()
}

func (e *LockError) Unwrap() error {
	return e.err
}

func (e *LockError) Is(target error) bool {
	if _, ok := target.(*LockError); ok {
		return true
	}
	return errors.Is(e.err, target)
}

type CheckConfigError struct {
	nothing any
	reason  string
	err     error
}

func NewCheckConfigError(nothing any, reason string, err error) *CheckConfigError {
	return &CheckConfigError{
		nothing: nothing,
		reason:  reason,
		err:     err,
	}
}

func (e *CheckConfigError) Error() string {
	return "[" +

		"example" +
		".User" +
		"] - " +
		"CheckConfig - " +
		e.reason +

		" - args: {" +

		"nothing: " + fmt.Sprintf("%#v", e.nothing) +
		"}" +

		"\n" +
		e.err.Error()
}

func (e *CheckConfigError) Unwrap() error {
	return e.err
}

func (e *CheckConfigError) Is(target error) bool {
	if _, ok := target.(*CheckConfigError); ok {
		return true
	}
	return errors.Is(e.err, target)
}
