package field

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/loader"
)

type BaseInfo struct {
	Name      string // name of field
	TypeName  string // name of type of field
	IsStruct  bool
	IsNumeric bool
	IsTime    bool
}

type Info struct {
	pointed *BaseInfo
	BaseInfo
	IsPointer bool
}

func (fi Info) GetPointed() Info {
	return Info{
		BaseInfo: *fi.pointed,
	}
}

type InfoGenerator struct {
	pkgInfo *loader.PackageInfo
}

type Field interface {
	Name() string
	Type() types.Type
}

type field struct {
	name string
	typ  types.Type
}

func (f field) Name() string {
	return f.name
}

func (f field) Type() types.Type {
	return f.typ
}

func NewInfoGenerator(pkgInfo *loader.PackageInfo) *InfoGenerator {
	return &InfoGenerator{
		pkgInfo: pkgInfo,
	}
}

func (g InfoGenerator) getOriginalTypeName(t *types.Named) string {
	if t.Obj().Pkg() == g.pkgInfo.Pkg {
		// t is from the same package as a struct
		return t.Obj().Name()
	}

	// t is an imported from another package type
	return fmt.Sprintf("%s.%s", t.Obj().Pkg().Name(), t.Obj().Name())
}

func (g InfoGenerator) GenFieldInfo(f Field) *Info { // nolint: interfacer
	typeName := f.Type().String()

	if typeName == "time.Time" {
		return &Info{
			BaseInfo: BaseInfo{
				Name:      f.Name(),
				TypeName:  typeName,
				IsNumeric: true,
				IsTime:    true,
			},
		}
	}

	switch t := f.Type().(type) {
	case *types.Basic:
		return &Info{
			BaseInfo: BaseInfo{
				Name:      f.Name(),
				TypeName:  typeName,
				IsNumeric: t.Info()&types.IsNumeric != 0,
			},
		}
	case *types.Named:
		r := g.GenFieldInfo(field{
			name: f.Name(),
			typ:  t.Underlying(),
		})
		if r != nil {
			r.TypeName = g.getOriginalTypeName(t)
		}
		return r
	case *types.Struct:
		return &Info{
			BaseInfo: BaseInfo{
				Name:     f.Name(),
				TypeName: typeName,
				IsStruct: true,
			},
		}
	case *types.Pointer:
		pf := g.GenFieldInfo(field{
			name: f.Name(),
			typ:  t.Elem(),
		})
		return &Info{
			BaseInfo: BaseInfo{
				Name:     f.Name(),
				TypeName: typeName,
			},
			IsPointer: true,
			pointed:   &pf.BaseInfo,
		}
	default:
		// no filtering is needed
		return nil
	}
}
