package example

type User struct {
	Name	string
	Age	int
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
		return NewProcessUserError(user, count, "user.UpdateName", err)
	}
	return NewProcessUserError(user, count, "processing failed", nil)
}
