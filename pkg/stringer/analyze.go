package stringer

import (
	"strings"

	"github.com/Bionic2113/errgen/pkg/utils"
	"github.com/dave/dst"
)

const skipValue = "-"

// MakeStringFuncs working with "type Name struct/Parent". For example
//
// With tags name "stringer":
//
//	type User struct {
//		Name string `stringer:"My name is"`
//		Age int `stringer:"I'm "`
//	}
//
//	func (u User) String() string {
//		return "My name is" + ":  " + u.Name +" "+ "I',m "+ ": " + strconv.Itoa(u.Age)
//	}
//
// Parent doesn't have String():
//
//	type OtherUser struct {
//		Name string
//		Age int
//	}
//
//	type Samuel OtherUser
//
//	func (s Samuel) String() string {
//		return "Name" + ": " + s.Name + " " + "Age" + ": " + strconv.Itoa(s.Age)
//	}
func (s *Stringer) MakeStringFuncs(pkgInfo utils.PkgInfo, scope *dst.Scope) {
	for k, v := range scope.Objects {
		if v.Decl == nil {
			continue
		}

		ts, ok := v.Decl.(*dst.TypeSpec)
		if !ok {
			continue
		}

		switch t := ts.Type.(type) {
		default:
			break // TODO:(bionic2113): для кастомных мап и тп некорректно сработает заполнение. Да и нужно ли
			// s.structsInfo[pkgInfo] = append(
			// 	s.structsInfo[pkgInfo],
			// 	StructInfo{Name: k, Fields: []*FieldInfo{{FactName: v.Name, Type: "any"}}},
			// )
		case *dst.StructType:
			structInfo, ok := s.makeStringFunc(k, t)
			if ok {
				s.structsInfo[pkgInfo] = append(s.structsInfo[pkgInfo], structInfo)
			}
		case *dst.Ident:
			// it is basic type
			if t.Obj == nil {
				break // TODO:(bionic2113): для кастомных базовых типов неккоректно сработает заполнение. Да и нужно ли
				// s.structsInfo[pkgInfo] = append(
				// 	s.structsInfo[pkgInfo],
				// 	StructInfo{Name: k, Fields: []*FieldInfo{{FactName: v.Name, Type: t.Name}}},
				// )
				// break
			}
			structInfo, ok := s.makeStringFunc(k, t.Obj.Decl.(*dst.TypeSpec).Type.(*dst.StructType))
			if ok {
				s.structsInfo[pkgInfo] = append(s.structsInfo[pkgInfo], structInfo)
			}
		}
	}
}

func (s *Stringer) makeStringFunc(name string, st *dst.StructType) (StructInfo, bool) {
	fields := make([]*FieldInfo, 0, len(st.Fields.List))
	for _, field := range st.Fields.List {
		tag, ok := s.tagValue(field)
		if !ok {
			continue
		}

		fieldInfo := &FieldInfo{
			CustomName: tag,
			Type:       "any", // that easier than real type for not basic type
		}

		fields = append(fields, fieldInfo)

		ident, ok := field.Type.(*dst.Ident)
		if !ok {
			fieldInfo.FactName = field.Names[0].Name
			continue
		}

		if utils.IsBasicType(ident.Name) {
			fieldInfo.Type = ident.Name
		}

		// Composition doesn't have names
		fieldInfo.FactName = ident.Name
		if len(field.Names) != 0 {
			fieldInfo.FactName = field.Names[0].Name
		}
	}

	if len(fields) == 0 {
		return StructInfo{}, false
	}

	return StructInfo{Name: name, Fields: fields}, true
}

func (s *Stringer) tagValue(field *dst.Field) (string, bool) {
	if field.Tag == nil {
		return "", true
	}

	tagString := strings.TrimSpace(field.Tag.Value)
	if len(tagString) == 0 {
		return "", true
	}

	if !strings.HasPrefix(tagString, "`") || !strings.HasSuffix(tagString, "`") {
		return "", true
	}

	tagString = tagString[1 : len(tagString)-1]

	parts := strings.Split(tagString, " ")
	for _, part := range parts {
		if !strings.HasPrefix(part, s.TagName+":\"") {
			continue
		}

		value := strings.TrimPrefix(part, s.TagName+":\"")
		value = value[:len(value)-1]

		if value == skipValue {
			return "", false
		}

		return value, true
	}

	return "", true
}
