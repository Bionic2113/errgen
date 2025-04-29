package example

// First comment
type User struct {
	Name string
	Age  int
}

func (u *User) UpdateName(newName string) error {
	if newName == "" {
		return NewUpdateNameError(newName, "unknown error in UpdateName", ErrExample1)
	}
	u.Name = newName

	return nil
}

func ProcessUser(user *User, count int) error {
	if err := user.UpdateName("New"); err != nil {
		return NewProcessUserError(user, count, "user.UpdateName", err)
	}

	return NewProcessUserError(user, count, "unknown error in ProcessUser", ErrExample2)
}

// Some comment
func (u *User) IsOlder(user *User, count int) (bool, error) {
	if user == nil {
		return false, NewIsOlderError(user, count, "unknown error in IsOlder", ErrExample3)
	}
	// Third comment
	if u == nil {
		return false, NewIsOlderError(user, count, "unknown error in IsOlder", ErrExample4)
	}
	if u.Age > user.Age {
		return true, nil
	}
	return false, nil
}

func (u *User) IsYounger(user *User, count int) (error, bool) {
	if user == nil {
		return NewIsYoungerError(user, count, "unknown error in IsYounger", ErrExample3), false
	}
	if u == nil {
		return NewIsYoungerError(user, count, "unknown error in IsYounger", ErrExample4), false
	}
	if u.Age < user.Age {
		return nil, true
	}
	return nil, false
}

// Last commnent
func (u *User) IsYoungerOrOlder(user *User, count int) (bool, bool, error) {
	if user == nil {
		return false, false, NewIsYoungerOrOlderError(user, count, "unknown error in IsYoungerOrOlder", ErrExample3)
	}
	if u == nil {
		return false, false, NewIsYoungerOrOlderError(user, count, "unknown error in IsYoungerOrOlder", ErrExample4)
	}
	if u.Age < user.Age {
		return true, false, nil
	}
	if u.Age > user.Age {
		return false, true, nil
	}
	return false, false, nil
}
