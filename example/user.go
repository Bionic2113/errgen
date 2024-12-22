package example

// First comment
type User struct {
	Name string
	Age  int
}

func (u *User) UpdateName(newName string) error {
	if newName == "" {
		return NewUpdateNameError(newName, "name cannot be empty", nil)
	}
	u.Name = newName

	return nil
}

func ProcessUser(user *User, count int) error {
	if err := user.UpdateName("New"); err != nil {
		return NewProcessUserError(user, count, "err", nil)
	}

	return NewProcessUserError(user, count, "processing failed", nil)
}

// Some comment
func (u *User) IsOlder(user *User, count int) (bool, error) {
	if user == nil {
		return false, NewIsOlderError(user, count, "user is nil", nil)
	}
	// Third comment
	if u == nil {
		return false, NewIsOlderError(user, count, "current user is nil", nil)
	}
	if u.Age > user.Age {
		return true, nil
	}
	return false, nil
}

func (u *User) IsYounger(user *User, count int) (error, bool) {
	if user == nil {
		return NewIsYoungerError(user, count, "user is nil", nil), false
	}
	if u == nil {
		return NewIsYoungerError(user, count, "current user is nil", nil), false
	}
	if u.Age < user.Age {
		return nil, true
	}
	return nil, false
}

// Last commnent
func (u *User) IsYoungerOrOlder(user *User, count int) (bool, bool, error) {
	if user == nil {
		return false, false, NewIsYoungerOrOlderError(user, count, "user is nil", nil)
	}
	if u == nil {
		return false, false, NewIsYoungerOrOlderError(user, count, "current user is nil", nil)
	}
	if u.Age < user.Age {
		return true, false, nil
	}
	if u.Age > user.Age {
		return false, true, nil
	}
	return false, false, nil
}
