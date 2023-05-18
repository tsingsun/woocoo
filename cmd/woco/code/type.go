package code

import (
	"fmt"
	"reflect"
	"strings"
)

// A Type represents a field type.
type Type uint8

// List of field types.
const (
	TypeInvalid Type = iota
	TypeBool
	TypeTime
	TypeJSON
	TypeUUID
	TypeBytes
	TypeEnum
	TypeString
	TypeOther
	TypeInt8
	TypeInt16
	TypeInt32
	TypeInt
	TypeInt64
	TypeUint8
	TypeUint16
	TypeUint32
	TypeUint
	TypeUint64
	TypeFloat32
	TypeFloat64
	endTypes
)

// String returns the string representation of a type.
func (t Type) String() string {
	if t < endTypes {
		return typeNames[t]
	}
	return typeNames[TypeInvalid]
}

// Numeric reports if the given type is a numeric type.
func (t Type) Numeric() bool {
	return t >= TypeInt8 && t < endTypes
}

// Float reports if the given type is a float type.
func (t Type) Float() bool {
	return t == TypeFloat32 || t == TypeFloat64
}

// Integer reports if the given type is an integral type.
func (t Type) Integer() bool {
	return t.Numeric() && !t.Float()
}

// Valid reports if the given type if known type.
func (t Type) Valid() bool {
	return t > TypeInvalid && t < endTypes
}

// ConstName returns the constant name of a info type.
// It's used by entc for printing the constant name in templates.
func (t Type) ConstName() string {
	switch {
	case !t.Valid():
		return typeNames[TypeInvalid]
	case int(t) < len(constNames) && constNames[t] != "":
		return constNames[t]
	default:
		return "Type" + strings.Title(typeNames[t])
	}
}

// TypeInfo holds the information regarding field type.
// Used by complex types like JSON and  Bytes.
type TypeInfo struct {
	Type     Type
	Ident    string
	PkgPath  string // import path.
	PkgName  string // local package name.
	Nillable bool   // slices,map or pointers.
	RType    *RType
}

// GoType parse by the given Type and reflect.Type. original field will override.
func (t *TypeInfo) GoType(typ any) {
	typeOf := reflect.TypeOf(typ)
	tv := Indirect(typeOf)
	info := TypeInfo{
		Type:    t.Type,
		Ident:   typeOf.String(),
		PkgPath: tv.PkgPath(),
		PkgName: PkgName(tv.String()),
		RType: &RType{
			rtype:   typeOf,
			Kind:    typeOf.Kind(),
			Name:    tv.Name(),
			Ident:   tv.String(),
			PkgPath: tv.PkgPath(),
			Methods: make(map[string]struct{ In, Out []*RType }, typeOf.NumMethod()),
		},
	}
	methods(typeOf, info.RType)
	switch typeOf.Kind() {
	case reflect.Slice, reflect.Ptr, reflect.Map:
		info.Nillable = true
	}
	*t = info
}

// String returns the string representation of a type.
func (t TypeInfo) String() string {
	switch {
	case t.Ident != "":
		return t.Ident
	case t.Type < endTypes:
		return typeNames[t.Type]
	default:
		return typeNames[TypeInvalid]
	}
}

func (t TypeInfo) StructString() string {
	s := t.String()
	if strings.HasPrefix(s, "*") {
		return s[1:]
	}
	return s
}

// Valid reports if the given type if known type.
func (t TypeInfo) Valid() bool {
	return t.Type.Valid()
}

// Numeric reports if the given type is a numeric type.
func (t TypeInfo) Numeric() bool {
	return t.Type.Numeric()
}

// ConstName returns the const name of the info type.
func (t TypeInfo) ConstName() string {
	return t.Type.ConstName()
}

// Comparable reports whether values of this type are comparable.
func (t TypeInfo) Comparable() bool {
	switch t.Type {
	case TypeBool, TypeTime, TypeUUID, TypeEnum, TypeString:
		return true
	case TypeOther:
		// Always accept custom types as comparable on the database side.
		// In the future, we should consider adding an interface to let
		// custom types tell if they are comparable or not (see #1304).
		return true
	default:
		return t.Numeric()
	}
}

// Clone returns a copy.
func (t TypeInfo) Clone() *TypeInfo {
	return &t
}

var (
	stringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
)

// Stringer indicates if this type implements the Stringer interface.
func (t TypeInfo) Stringer() bool {
	return t.RType.implements(stringerType)
}

var (
	typeNames = [...]string{
		TypeInvalid: "invalid",
		TypeBool:    "bool",
		TypeTime:    "time.Time",
		TypeJSON:    "json.RawMessage",
		TypeUUID:    "[16]byte",
		TypeBytes:   "[]byte",
		TypeEnum:    "string",
		TypeString:  "string",
		TypeOther:   "other",
		TypeInt:     "int",
		TypeInt8:    "int8",
		TypeInt16:   "int16",
		TypeInt32:   "int32",
		TypeInt64:   "int64",
		TypeUint:    "uint",
		TypeUint8:   "uint8",
		TypeUint16:  "uint16",
		TypeUint32:  "uint32",
		TypeUint64:  "uint64",
		TypeFloat32: "float32",
		TypeFloat64: "float64",
	}
	constNames = [...]string{
		TypeJSON:  "TypeJSON",
		TypeUUID:  "TypeUUID",
		TypeTime:  "TypeTime",
		TypeEnum:  "TypeEnum",
		TypeBytes: "TypeBytes",
		TypeOther: "TypeOther",
	}
)

// RType holds a serializable reflect.Type information of
// Go object. Used by the entc package.
type RType struct {
	Name    string // reflect.Type.Name
	Ident   string // reflect.Type.String
	Kind    reflect.Kind
	PkgPath string
	Methods map[string]struct{ In, Out []*RType }
	// Used only for in-package checks.
	rtype reflect.Type
}

// TypeEqual reports if the underlying type is equal to the RType (after pointer indirections).
func (r *RType) TypeEqual(t reflect.Type) bool {
	tv := Indirect(t)
	return r.Name == tv.Name() && r.Kind == t.Kind() && r.PkgPath == tv.PkgPath()
}

// RType returns the string value of the Indirect reflect.Type.
func (r *RType) String() string {
	if r.rtype != nil {
		return r.rtype.String()
	}
	return r.Ident
}

// IsPtr reports if the reflect-type is a pointer type.
func (r *RType) IsPtr() bool {
	return r != nil && r.Kind == reflect.Ptr
}

func (r *RType) implements(typ reflect.Type) bool {
	if r == nil {
		return false
	}
	n := typ.NumMethod()
	for i := 0; i < n; i++ {
		m0 := typ.Method(i)
		m1, ok := r.Methods[m0.Name]
		if !ok || len(m1.In) != m0.Type.NumIn() || len(m1.Out) != m0.Type.NumOut() {
			return false
		}
		in := m0.Type.NumIn()
		for j := 0; j < in; j++ {
			if !m1.In[j].TypeEqual(m0.Type.In(j)) {
				return false
			}
		}
		out := m0.Type.NumOut()
		for j := 0; j < out; j++ {
			if !m1.Out[j].TypeEqual(m0.Type.Out(j)) {
				return false
			}
		}
	}
	return true
}

func (r RType) ReflectType() reflect.Type {
	return r.rtype
}

func ParseGoType(typ any) (*RType, error) {
	t := reflect.TypeOf(typ)
	tv := Indirect(t)
	rt := &RType{
		rtype:   t,
		Name:    tv.Name(),
		Kind:    t.Kind(),
		PkgPath: tv.PkgPath(),
		Ident:   tv.String(),
		Methods: make(map[string]struct{ In, Out []*RType }, t.NumMethod()),
	}
	methods(t, rt)
	return rt, nil
}

func methods(t reflect.Type, rtype *RType) {
	// For type T, add methods with
	// pointer receiver as well (*T).
	if t.Kind() != reflect.Ptr {
		t = reflect.PtrTo(t)
	}
	n := t.NumMethod()
	for i := 0; i < n; i++ {
		m := t.Method(i)
		in := make([]*RType, m.Type.NumIn()-1)
		for j := range in {
			arg := m.Type.In(j + 1)
			in[j] = &RType{Name: arg.Name(), Ident: arg.String(), Kind: arg.Kind(), PkgPath: arg.PkgPath()}
		}
		out := make([]*RType, m.Type.NumOut())
		for j := range out {
			ret := m.Type.Out(j)
			out[j] = &RType{Name: ret.Name(), Ident: ret.String(), Kind: ret.Kind(), PkgPath: ret.PkgPath()}
		}
		rtype.Methods[m.Name] = struct{ In, Out []*RType }{in, out}
	}
}
