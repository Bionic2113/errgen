package example

type OtherUser struct {
	Compos
	Name  string
	token string
	Phone Phone
	Age   uint8
}

type Phone struct {
	Type    string
	Number  string
	imei    string `errgen:"skip"`
	isSmart bool   `errgen:"-"`
}

type Igor OtherUser

type Compos struct {
	One int
	Two int
}

type AnyCheck struct {
	MyyMap map[int]int
	MyArr  []any
	any
	Interface interface{}
	Foo       func(argOne string) bool
}

type MyMap map[int]int
type MyString string
