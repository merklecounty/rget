package analysisutil

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var errType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

// ImplementsError return whether t implements error interface.
func ImplementsError(t types.Type) bool {
	return types.Implements(t, errType)
}

// ObjectOf returns types.Object by given name in the package.
func ObjectOf(pass *analysis.Pass, pkg, name string) types.Object {
	return LookupFromImports(pass.Pkg.Imports(), pkg, name)
}

// TypeOf returns types.Type by given name in the package.
// TypeOf accepts pointer types such as *T.
func TypeOf(pass *analysis.Pass, pkg, name string) types.Type {
	if name == "" {
		return nil
	}

	if name[0] == '*' {
		return types.NewPointer(TypeOf(pass, pkg, name[1:]))
	}

	obj := ObjectOf(pass, pkg, name)
	if obj == nil {
		return nil
	}

	return obj.Type()
}

// MethodOf returns a method which has given name in the type.
func MethodOf(typ types.Type, name string) *types.Func {
	switch typ := typ.(type) {
	case *types.Named:
		for i := 0; i < typ.NumMethods(); i++ {
			if f := typ.Method(i); f.Id() == name {
				return f
			}
		}
	case *types.Pointer:
		return MethodOf(typ.Elem(), name)
	}
	return nil
}

// see: https://github.com/golang/go/issues/19670
func identical(x, y types.Type) (ret bool) {
	defer func() {
		r := recover()
		switch r := r.(type) {
		case string:
			if r == "unreachable" {
				ret = false
				return
			}
		case nil:
			return
		}
		panic(r)
	}()
	return types.Identical(x, y)
}

// Interfaces returns a map of interfaces which are declared in the package.
func Interfaces(pkg *types.Package) map[string]*types.Interface {
	ifs := map[string]*types.Interface{}

	for _, n := range pkg.Scope().Names() {
		o := pkg.Scope().Lookup(n)
		if o != nil {
			i, ok := o.Type().Underlying().(*types.Interface)
			if ok {
				ifs[n] = i
			}
		}
	}

	return ifs
}

// Structs returns a map of structs which are declared in the package.
func Structs(pkg *types.Package) map[string]*types.Struct {
	structs := map[string]*types.Struct{}

	for _, n := range pkg.Scope().Names() {
		o := pkg.Scope().Lookup(n)
		if o != nil {
			s, ok := o.Type().Underlying().(*types.Struct)
			if ok {
				structs[n] = s
			}
		}
	}

	return structs
}

// HasField returns whether the struct has the field.
func HasField(s *types.Struct, f *types.Var) bool {
	if s == nil || f == nil {
		return false
	}

	for i := 0; i < s.NumFields(); i++ {
		if s.Field(i) == f {
			return true
		}
	}

	return false
}
