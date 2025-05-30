package example

import (
	"encoding/json"
)

type SomeStruct struct {
	Name string
	Age  int
}

func WithAnon() error {
	f := func() bool {
		return true
	}

	_ = f
	return nil
}

func WithAnon_2() error {
	func() bool {
		return true
	}()

	return nil
}

func WithAnon_3() error {
	func() bool {
		_, err := Marshal()
		if err != nil {
			return false
		}
		if _, err := Marshal(); err != nil {
			return false
		}
		return true
	}()

	func() error {
		_, err := Marshal()
		if err != nil {
			return err
		}
		if _, err := Marshal(); err != nil {
			return err
		}
		return nil
	}()

	return nil
}

func Marshal() ([]byte, error) {
	ss := SomeStruct{Name: "Oleg", Age: 164}
	return json.Marshal(ss)
}
