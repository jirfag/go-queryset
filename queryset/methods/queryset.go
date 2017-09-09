package methods

import (
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/jinzhu/gorm"
)

// retQuerySetMethod

type retQuerySetMethod struct {
	qsTypeName string
}

// GetReturnValuesDeclaration gets return values declaration
func (m retQuerySetMethod) GetReturnValuesDeclaration() string {
	return m.qsTypeName
}

func newRetQuerySetMethod(qsTypeName string) retQuerySetMethod {
	return retQuerySetMethod{
		qsTypeName: qsTypeName,
	}
}

// baseQuerySetMethod

type baseQuerySetMethod struct {
	structMethod
}

func newBaseQuerySetMethod(qsTypeName string) baseQuerySetMethod {
	return baseQuerySetMethod{
		structMethod: newStructMethod("qs", qsTypeName),
	}
}

// FieldOperationNoArgsMethod is for unary operations: preload, orderby, etc
type FieldOperationNoArgsMethod struct {
	callGormMethod
	onFieldMethod
	noArgsMethod
	baseQuerySetMethod
	retQuerySetMethod
}

// GetBody returns body
func (m FieldOperationNoArgsMethod) GetBody() string {
	return wrapToGormScope(m.callGormMethod.GetBody())
}

func newFieldOperationNoArgsMethod(name, fieldName, qsTypeName string,
	transformFieldName bool) FieldOperationNoArgsMethod {

	gormArgName := fieldName
	if transformFieldName {
		gormArgName = gorm.ToDBName(fieldName)
	}
	gormArgName = fmt.Sprintf(`"%s"`, gormArgName)

	r := FieldOperationNoArgsMethod{
		onFieldMethod:      newOnFieldMethod(name, fieldName),
		callGormMethod:     newCallGormMethod(name, gormArgName, "d"),
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		retQuerySetMethod:  newRetQuerySetMethod(qsTypeName),
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
	return wrapToGormScope(fmt.Sprintf(`return d.%s(%s)`, m.name, m.getArgName()))
}

// LowercaseFirstRune lowercases first rune of string
func LowercaseFirstRune(s string) string {
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func fieldNameToArgName(fieldName string) string {
	if fieldName == "ID" {
		return fieldName
	}

	return LowercaseFirstRune(fieldName)
}

func newFieldOperationOneArgMethod(name, fieldName, argTypeName string) fieldOperationOneArgMethod {
	return fieldOperationOneArgMethod{
		onFieldMethod: newOnFieldMethod(name, fieldName),
		oneArgMethod:  newOneArgMethod(fieldNameToArgName(fieldName), argTypeName),
	}
}

// StructOperationOneArgMethod is for struct operations with one arg
type StructOperationOneArgMethod struct {
	namedMethod
	baseQuerySetMethod
	retQuerySetMethod
	oneArgMethod
}

// GetBody returns method body
func (m StructOperationOneArgMethod) GetBody() string {
	return wrapToGormScope(fmt.Sprintf(`return d.%s(%s)`, m.name, m.getArgName()))
}

func newStructOperationOneArgMethod(name, argTypeName, qsTypeName string) StructOperationOneArgMethod {
	return StructOperationOneArgMethod{
		namedMethod:        newNamedMethod(name),
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		retQuerySetMethod:  newRetQuerySetMethod(qsTypeName),
		oneArgMethod:       newOneArgMethod(strings.ToLower(name), argTypeName),
	}
}

// BinaryFilterMethod is a binary filter method
type BinaryFilterMethod struct {
	fieldOperationOneArgMethod
	baseQuerySetMethod
	retQuerySetMethod
}

// NewBinaryFilterMethod create new binary filter method
func NewBinaryFilterMethod(name, fieldName, argTypeName, qsTypeName string) BinaryFilterMethod {
	return BinaryFilterMethod{
		fieldOperationOneArgMethod: newFieldOperationOneArgMethod(
			name, fieldName, argTypeName),
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		retQuerySetMethod:  newRetQuerySetMethod(qsTypeName),
	}
}

// GetBody returns method's code
func (m BinaryFilterMethod) GetBody() string {
	return wrapToGormScope(fmt.Sprintf(`return d.Where("%s %s", %s)`,
		gorm.ToDBName(m.fieldName), m.getWhereCondition(), m.getArgName()))
}

func (m BinaryFilterMethod) getWhereCondition() string {
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

// UnaryFilterMethod represents unary filter
type UnaryFilterMethod struct {
	onFieldMethod
	noArgsMethod
	baseQuerySetMethod
	retQuerySetMethod
	op string
}

func newUnaryFilterMethod(name, fieldName, op, qsTypeName string) UnaryFilterMethod {
	r := UnaryFilterMethod{
		onFieldMethod:      newOnFieldMethod(name, fieldName),
		op:                 op,
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		retQuerySetMethod:  newRetQuerySetMethod(qsTypeName),
	}
	r.setFieldNameFirst(true)
	return r
}

// GetBody returns method's code
func (m UnaryFilterMethod) GetBody() string {
	return wrapToGormScope(fmt.Sprintf(`return d.Where("%s %s")`,
		gorm.ToDBName(m.fieldName), m.op))
}

// unaryFilerMethod

// SelectMethod is a select field (all, one, etc)
type SelectMethod struct {
	namedMethod
	oneArgMethod
	baseQuerySetMethod
	gormErroredMethod
}

func newSelectMethod(name, gormName, argTypeName, qsTypeName string) SelectMethod {
	return SelectMethod{
		namedMethod:        newNamedMethod(name),
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		oneArgMethod:       newOneArgMethod("ret", argTypeName),
		gormErroredMethod:  newGormErroredMethod(gormName, "ret", "qs.db"),
	}
}

// GetUpdaterMethod creates GetUpdater method
type GetUpdaterMethod struct {
	baseQuerySetMethod
	namedMethod
	noArgsMethod
	constRetMethod
	constBodyMethod
}

// NewGetUpdaterMethod creates GetUpdaterMethod
func NewGetUpdaterMethod(qsTypeName, updaterTypeMethod string) GetUpdaterMethod {
	return GetUpdaterMethod{
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		namedMethod:        newNamedMethod("GetUpdater"),
		constRetMethod:     newConstRetMethod(updaterTypeMethod),
		constBodyMethod:    newConstBodyMethod("return New%s(qs.db)", updaterTypeMethod),
	}
}

// DeleteMethod creates Delete method
type DeleteMethod struct {
	baseQuerySetMethod
	namedMethod
	noArgsMethod
	errorRetMethod
	constBodyMethod
}

// NewDeleteMethod creates Delete method
func NewDeleteMethod(qsTypeName, structTypeName string) DeleteMethod {
	cbm := newConstBodyMethod("return qs.db.Delete(%s{}).Error", structTypeName)
	return DeleteMethod{
		baseQuerySetMethod: newBaseQuerySetMethod(qsTypeName),
		namedMethod:        newNamedMethod("Delete"),
		constBodyMethod:    cbm,
	}
}

// Concrete methods

// NewPreloadMethod creates new Preload method
func NewPreloadMethod(fieldName, qsTypeName string) FieldOperationNoArgsMethod {
	r := newFieldOperationNoArgsMethod("Preload", fieldName, qsTypeName, false)
	return r
}

// NewOrderByMethod creates new OrderBy method
func NewOrderByMethod(fieldName, qsTypeName string) FieldOperationNoArgsMethod {
	r := newFieldOperationNoArgsMethod("OrderBy", fieldName, qsTypeName, true)
	r.setGormMethodName("Order")
	return r
}

// NewLimitMethod creates Limit method
func NewLimitMethod(qsTypeName string) StructOperationOneArgMethod {
	return newStructOperationOneArgMethod("Limit", "int", qsTypeName)
}

// NewAllMethod creates All method
func NewAllMethod(structName, qsTypeName string) SelectMethod {
	return newSelectMethod("All", "Find", fmt.Sprintf("*[]%s", structName), qsTypeName)
}

// NewOneMethod creates One method
func NewOneMethod(structName, qsTypeName string) SelectMethod {
	r := newSelectMethod("One", "First", fmt.Sprintf("*%s", structName), qsTypeName)
	const doc = `// One is used to retrieve one result. It returns gorm.ErrRecordNotFound
	// if nothing was fetched`
	r.setDoc(doc)
	return r
}

// NewIsNullMethod create IsNull method
func NewIsNullMethod(fieldName, qsTypeName string) UnaryFilterMethod {
	return newUnaryFilterMethod("IsNull", fieldName, "IS NULL", qsTypeName)
}
