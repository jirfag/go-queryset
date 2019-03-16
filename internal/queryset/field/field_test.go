package field

import (
	"go/token"
	"go/types"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type tf struct {
	name string
	typ  types.Type
	tag  reflect.StructTag
}

func (f tf) Name() string           { return f.name }
func (f tf) Type() types.Type       { return f.typ }
func (f tf) Tag() reflect.StructTag { return f.tag }

func newTf(name string, typ types.Type, tag string) tf {
	return tf{
		name: name,
		typ:  typ,
		tag:  reflect.StructTag(tag),
	}
}

func newG() *InfoGenerator {
	return NewInfoGenerator(nil)
}

func genFieldInfo(f Field) *Info {
	return newG().GenFieldInfo(f)
}

var typeString = types.Typ[types.String]
var typeStringPtr = types.NewPointer(typeString)
var typeNamedString = types.NewNamed(
	types.NewTypeName(token.Pos(0), nil, "myNamedType", typeString),
	typeString,
	nil)

const fName = "F"

func TestIgnoredByTagColumn(t *testing.T) {
	const (
		gormIgnore = `gorm:"-"`
		sqlIgnore  = `sql:"-"`
	)

	assert.Nil(t, genFieldInfo(newTf(fName, typeString, gormIgnore)))
	assert.Nil(t, genFieldInfo(newTf(fName, typeStringPtr, gormIgnore)))
	assert.Nil(t, genFieldInfo(newTf(fName, typeString, sqlIgnore)))
}

func TestColumnNameSetInTag(t *testing.T) {
	const colNameZ = `gorm:"column:z"`

	info := genFieldInfo(newTf(fName, typeString, colNameZ))
	assert.Equal(t, "z", info.DBName)
	assert.Equal(t, fName, info.Name)
	assert.Equal(t, typeString.String(), info.TypeName)

	info = genFieldInfo(newTf(fName, typeStringPtr, colNameZ))
	assert.Equal(t, "z", info.DBName)
	assert.Equal(t, fName, info.Name)
	assert.Equal(t, typeStringPtr.String(), info.TypeName)

	info = genFieldInfo(newTf(fName, typeNamedString, colNameZ))
	assert.Equal(t, "z", info.DBName)
	assert.Equal(t, fName, info.Name)
	assert.Equal(t, typeNamedString.String(), info.TypeName)
}
