package methods

import (
	"strings"
)

// baseUpdaterMethod
type baseUpdaterMethod struct {
	structMethod
	updaterTypeName string
}

func newBaseUpdaterMethod(updaterTypeName string) baseUpdaterMethod {
	return baseUpdaterMethod{
		updaterTypeName: updaterTypeName,
		structMethod:    newStructMethod("u", updaterTypeName),
	}
}

// UpdaterSetMethod generates Set<Field> method
type UpdaterSetMethod struct {
	onFieldMethod
	oneArgMethod
	baseUpdaterMethod
	constRetMethod
	constBodyMethod

	dbSchemaTypeName string
}

// NewUpdaterSetMethod create new SetField method
func NewUpdaterSetMethod(fieldName, fieldTypeName,
	updaterTypeName, dbSchemaTypeName string) UpdaterSetMethod {

	argName := fieldNameToArgName(fieldName)
	cbm := newConstBodyMethod(
		`u.fields[string(%s.%s)] = %s
		return u`,
		dbSchemaTypeName,
		fieldName,
		argName)

	r := UpdaterSetMethod{
		onFieldMethod:     newOnFieldMethod("Set", fieldName),
		oneArgMethod:      newOneArgMethod(argName, fieldTypeName),
		baseUpdaterMethod: newBaseUpdaterMethod(updaterTypeName),
		constRetMethod:    newConstRetMethod(updaterTypeName),
		constBodyMethod:   cbm,
		dbSchemaTypeName:  dbSchemaTypeName,
	}
	r.setFieldNameFirst(false)
	return r
}

// UpdaterUpdateMethod creates Update method
type UpdaterUpdateMethod struct {
	namedMethod
	baseUpdaterMethod
	noArgsMethod
	errorRetMethod
	constBodyMethod
}

// NewUpdaterUpdateMethod create new Update method
func NewUpdaterUpdateMethod(updaterTypeName string) UpdaterUpdateMethod {
	return UpdaterUpdateMethod{
		namedMethod:       newNamedMethod("Update"),
		baseUpdaterMethod: newBaseUpdaterMethod(updaterTypeName),
		constBodyMethod:   newConstBodyMethod("return u.db.Updates(u.fields).Error"),
	}
}

// UpdaterUpdateNumMethod describes UpdateNum method
type UpdaterUpdateNumMethod struct {
	namedMethod
	baseUpdaterMethod
	noArgsMethod
	constRetMethod
	constBodyMethod
}

// NewUpdaterUpdateNumMethod creates new UpdateNum method
func NewUpdaterUpdateNumMethod(updaterTypeName string) UpdaterUpdateNumMethod {
	return UpdaterUpdateNumMethod{
		namedMethod:       newNamedMethod("UpdateNum"),
		baseUpdaterMethod: newBaseUpdaterMethod(updaterTypeName),
		constRetMethod:    newConstRetMethod("(int64, error)"),
		constBodyMethod: newConstBodyMethod(
			strings.Join([]string{
				"db := u.db.Updates(u.fields)",
				"return db.RowsAffected, db.Error",
			}, "\n"),
		),
	}
}
