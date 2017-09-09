package methods

// StructModifierMethod represents method, modifying current struct
type StructModifierMethod struct {
	namedMethod
	structMethod
	dbArgMethod
	gormErroredMethod
}

// NewStructModifierMethod create StructModifierMethod method
func NewStructModifierMethod(name, structTypeName string) StructModifierMethod {
	r := StructModifierMethod{
		namedMethod:       newNamedMethod(name),
		dbArgMethod:       newDbArgMethod(),
		structMethod:      newStructMethod("o", "*"+structTypeName),
		gormErroredMethod: newGormErroredMethod(name, "o", "db"),
	}
	return r
}
