package field

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"

	"gorm.io/gorm/schema"
)

type BaseInfo struct {
	Name      string // name of field
	DBName    string // name of field in DB
	TypeName  string // name of type of field
	IsStruct  bool
	IsNumeric bool
	IsTime    bool
	IsString  bool
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
	pkg *types.Package
}

type Field interface {
	Name() string
	Type() types.Type
	Tag() reflect.StructTag
}

type field struct {
	name string
	typ  types.Type
	tag  reflect.StructTag
}

func (f field) Name() string {
	return f.name
}

func (f field) Type() types.Type {
	return f.typ
}

func (f field) Tag() reflect.StructTag {
	return f.tag
}

func NewInfoGenerator(pkg *types.Package) *InfoGenerator {
	return &InfoGenerator{
		pkg: pkg,
	}
}

func (g InfoGenerator) getOriginalTypeName(t *types.Named) string {
	if t.Obj().Pkg() == g.pkg {
		// t is from the same package as a struct
		return t.Obj().Name()
	}

	// t is an imported from another package type
	return fmt.Sprintf("%s.%s", t.Obj().Pkg().Name(), t.Obj().Name())
}

// parseTagSetting is copy-pasted from gorm source code.
func parseTagSetting(tags reflect.StructTag) map[string]string {
	setting := map[string]string{}
	for _, str := range []string{tags.Get("sql"), tags.Get("gorm")} {
		tags := strings.Split(str, ";")
		for _, value := range tags {
			v := strings.Split(value, ":")
			k := strings.TrimSpace(strings.ToUpper(v[0]))
			if len(v) >= 2 {
				setting[k] = strings.Join(v[1:], ":")
			} else {
				setting[k] = k
			}
		}
	}
	return setting
}

func (g InfoGenerator) GenFieldInfo(f Field) *Info {
	tagSetting := parseTagSetting(f.Tag())
	if tagSetting["-"] != "" { // skipped by tag field
		return nil
	}

	dbName := schema.NamingStrategy{}.ColumnName("", f.Name())
	if dbColName := tagSetting["COLUMN"]; dbColName != "" {
		dbName = dbColName
	}
	bi := BaseInfo{
		Name:     f.Name(),
		TypeName: f.Type().String(),
		DBName:   dbName,
	}

	if bi.TypeName == "time.Time" {
		bi.IsTime = true
		bi.IsNumeric = true
		return &Info{
			BaseInfo: bi,
		}
	}

	switch t := f.Type().(type) {
	case *types.Basic:
		bi.IsString = t.Info()&types.IsString != 0
		bi.IsNumeric = t.Info()&types.IsNumeric != 0
		return &Info{
			BaseInfo: bi,
		}
	case *types.Slice:
		if t.Elem().String() == "byte" {
			return &Info{
				BaseInfo: bi,
			}
		}
		return nil
	case *types.Named:
		r := g.GenFieldInfo(field{
			name: f.Name(),
			typ:  t.Underlying(),
			tag:  f.Tag(),
		})
		if r != nil {
			r.TypeName = g.getOriginalTypeName(t)
		}
		return r
	case *types.Struct:
		bi.IsStruct = true
		return &Info{
			BaseInfo: bi,
		}
	case *types.Pointer:
		pf := g.GenFieldInfo(field{
			name: f.Name(),
			typ:  t.Elem(),
			tag:  f.Tag(),
		})
		return &Info{
			BaseInfo:  bi,
			IsPointer: true,
			pointed:   &pf.BaseInfo,
		}
	default:
		// no filtering is needed
		return nil
	}
}
