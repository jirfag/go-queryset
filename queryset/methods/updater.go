package methods

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
