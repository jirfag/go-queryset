package queryset

import (
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/jinzhu/gorm"
)

type method interface {
	GetMethodName() string
	GetArgsDeclaration() string
	GetBody() string
}

// baseMethod

type baseMethod struct {
	name string
}

func newBaseMethod(name string) baseMethod {
	return baseMethod{
		name: name,
	}
}

// GetMethodName returns name of method
func (m baseMethod) GetMethodName() string {
	return m.name
}

func (m baseMethod) wrapMethod(code string) string {
	const tmpl = `qs.db = qs.db.Scopes(func(d *gorm.DB) *gorm.DB {
      %s})
    return qs`
	return fmt.Sprintf(tmpl, code)
}

// onFieldMethod

type onFieldMethod struct {
	baseMethod
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
		baseMethod:       newBaseMethod(name),
		fieldName:        fieldName,
		isFieldNameFirst: true,
	}
}

// oneArgMethod

type oneArgMethod struct {
	argName     string
	argTypeName string
}

func (m oneArgMethod) getArgName() string {
	return m.argName
}

// GetArgsDeclaration returns declaration of arguments list for func decl
func (m oneArgMethod) GetArgsDeclaration() string {
	return fmt.Sprintf("%s %s", m.getArgName(), m.argTypeName)
}

func newOneArgMethod(argName, argTypeName string) oneArgMethod {
	return oneArgMethod{
		argName:     argName,
		argTypeName: argTypeName,
	}
}

// noArgsMethod

type noArgsMethod struct{}

// GetArgsDeclaration returns declaration of arguments list for func decl
func (m noArgsMethod) GetArgsDeclaration() string {
	return ""
}

// fieldOperationNoArgsMethod

// fieldOperationNoArgsMethod is for unary operations: preload, orderby, etc
type fieldOperationNoArgsMethod struct {
	onFieldMethod
	noArgsMethod
	gormMethodName string
}

func (m *fieldOperationNoArgsMethod) setGormMethodName(name string) {
	m.gormMethodName = name
}

// GetBody returns method body
func (m fieldOperationNoArgsMethod) GetBody() string {
	return m.wrapMethod(fmt.Sprintf(`return d.%s("%s")`, m.gormMethodName, gorm.ToDBName(m.fieldName)))
}

func newFieldOperationNoArgsMethod(name, fieldName string) fieldOperationNoArgsMethod {
	r := fieldOperationNoArgsMethod{
		onFieldMethod:  newOnFieldMethod(name, fieldName),
		gormMethodName: name,
	}
	r.setFieldNameFirst(false) // UserPreload -> PreloadUser
	return r
}

// fieldOperationOneArgMethod

type fieldOperationOneArgMethod struct {
	onFieldMethod
	oneArgMethod
}

// GetBody returns method body
func (m fieldOperationOneArgMethod) GetBody() string {
	return m.wrapMethod(fmt.Sprintf(`return d.%s(%s)`, m.name, m.getArgName()))
}

func lowercaseFirstRune(s string) string {
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func fieldNameToArgName(fieldName string) string {
	if fieldName == "ID" {
		return fieldName
	}

	return lowercaseFirstRune(fieldName)
}

func newFieldOperationOneArgMethod(name, fieldName, argTypeName string) fieldOperationOneArgMethod {
	return fieldOperationOneArgMethod{
		onFieldMethod: newOnFieldMethod(name, fieldName),
		oneArgMethod:  newOneArgMethod(fieldNameToArgName(fieldName), argTypeName),
	}
}

// structOperationOneArgMethod

type structOperationOneArgMethod struct {
	baseMethod
	oneArgMethod
}

// GetBody returns method body
func (m structOperationOneArgMethod) GetBody() string {
	return m.wrapMethod(fmt.Sprintf(`return d.%s(%s)`, m.name, m.getArgName()))
}

func newStructOperationOneArgMethod(name, argTypeName string) structOperationOneArgMethod {
	return structOperationOneArgMethod{
		baseMethod:   newBaseMethod(name),
		oneArgMethod: newOneArgMethod(strings.ToLower(name), argTypeName),
	}
}

// binaryFilterMethod

type binaryFilterMethod struct {
	fieldOperationOneArgMethod
}

func newBinaryFilterMethod(name, fieldName, argTypeName string) binaryFilterMethod {
	return binaryFilterMethod{
		fieldOperationOneArgMethod: newFieldOperationOneArgMethod(name, fieldName, argTypeName),
	}
}

// GetBody returns method's code
func (m binaryFilterMethod) GetBody() string {
	return m.wrapMethod(fmt.Sprintf(`return d.Where("%s %s", %s)`,
		gorm.ToDBName(m.fieldName), m.getWhereCondition(), m.getArgName()))
}

func (m binaryFilterMethod) getWhereCondition() string {
	nameToOp := map[string]string{
		"eq":  "=",
		"ne":  "!=",
		"lt":  "<",
		"lte": "<=",
		"gt":  ">",
		"gte": ">=",
	}
	op := nameToOp[m.name]
	if op == "" {
		log.Fatalf("no operation for filter %q", m.name)
	}

	return fmt.Sprintf("%s ?", op)
}

// Concrete methods

func newPreloadMethod(fieldName string) fieldOperationNoArgsMethod {
	return newFieldOperationNoArgsMethod("Preload", fieldName)
}

func newOrderByMethod(fieldName string) fieldOperationNoArgsMethod {
	r := newFieldOperationNoArgsMethod("OrderBy", fieldName)
	r.setGormMethodName("Order")
	return r
}

func newLimitMethod() structOperationOneArgMethod {
	return newStructOperationOneArgMethod("Limit", "int")
}
