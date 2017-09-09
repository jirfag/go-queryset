package methods

import "strings"

// onFieldMethod

type onFieldMethod struct {
	namedMethod
	fieldName        string
	isFieldNameFirst bool
}

func (m *onFieldMethod) setFieldNameFirst(isFieldNameFirst bool) {
	m.isFieldNameFirst = isFieldNameFirst
}

// GetMethodName returns name of method
func (m onFieldMethod) GetMethodName() string {
	args := []string{m.fieldName, strings.Title(m.name)}
	if !m.isFieldNameFirst {
		args[0], args[1] = args[1], args[0]
	}
	return args[0] + args[1]
}

func newOnFieldMethod(name, fieldName string) onFieldMethod {
	return onFieldMethod{
		namedMethod:      newNamedMethod(name),
		fieldName:        fieldName,
		isFieldNameFirst: true,
	}
}
