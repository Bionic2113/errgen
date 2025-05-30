package example

import (
	"errors"
	"fmt"
	"strconv"
)

type WithAnonError struct {
	reason string
	err    error
}

func NewWithAnonError(reason string, err error) *WithAnonError {
	return &WithAnonError{
		reason: reason,
		err:    err,
	}
}

func (e *WithAnonError) Error() string {
	return "[" +

		"example" +

		"] - " +
		"WithAnon - " +
		e.reason +

		"\n" +
		e.err.Error()
}

func (e *WithAnonError) Unwrap() error {
	return e.err
}

func (e *WithAnonError) Is(target error) bool {
	if _, ok := target.(*WithAnonError); ok {
		return true
	}
	return errors.Is(e.err, target)
}

type WithAnon_2Error struct {
	reason string
	err    error
}

func NewWithAnon_2Error(reason string, err error) *WithAnon_2Error {
	return &WithAnon_2Error{
		reason: reason,
		err:    err,
	}
}

func (e *WithAnon_2Error) Error() string {
	return "[" +

		"example" +

		"] - " +
		"WithAnon_2 - " +
		e.reason +

		"\n" +
		e.err.Error()
}

func (e *WithAnon_2Error) Unwrap() error {
	return e.err
}

func (e *WithAnon_2Error) Is(target error) bool {
	if _, ok := target.(*WithAnon_2Error); ok {
		return true
	}
	return errors.Is(e.err, target)
}

type WithAnon_3Error struct {
	reason string
	err    error
}

func NewWithAnon_3Error(reason string, err error) *WithAnon_3Error {
	return &WithAnon_3Error{
		reason: reason,
		err:    err,
	}
}

func (e *WithAnon_3Error) Error() string {
	return "[" +

		"example" +

		"] - " +
		"WithAnon_3 - " +
		e.reason +

		"\n" +
		e.err.Error()
}

func (e *WithAnon_3Error) Unwrap() error {
	return e.err
}

func (e *WithAnon_3Error) Is(target error) bool {
	if _, ok := target.(*WithAnon_3Error); ok {
		return true
	}
	return errors.Is(e.err, target)
}

type MarshalError struct {
	reason string
	err    error
}

func NewMarshalError(reason string, err error) *MarshalError {
	return &MarshalError{
		reason: reason,
		err:    err,
	}
}

func (e *MarshalError) Error() string {
	return "[" +

		"example" +

		"] - " +
		"Marshal - " +
		e.reason +

		"\n" +
		e.err.Error()
}

func (e *MarshalError) Unwrap() error {
	return e.err
}

func (e *MarshalError) Is(target error) bool {
	if _, ok := target.(*MarshalError); ok {
		return true
	}
	return errors.Is(e.err, target)
}

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
