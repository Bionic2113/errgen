package example

import (
	"fmt"
	"strconv"
)


type UpdateNameError struct {
	newName string
	reason string
	err    error
}

func NewUpdateNameError(newName string, reason string, err error) *UpdateNameError {
	return &UpdateNameError{
		newName: newName,
		reason: reason,
		err:    err,
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

type ProcessUserError struct {
	user *User
	count int
	reason string
	err    error
}

func NewProcessUserError(user *User, count int, reason string, err error) *ProcessUserError {
	return &ProcessUserError{
		user: user,
		count: count,
		reason: reason,
		err:    err,
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
