package methods

import "fmt"

func wrapToGormScope(code string) string {
	const tmpl = `qs.db = qs.db.Scopes(func(d *gorm.DB) *gorm.DB {
      %s})
    return qs`
	return fmt.Sprintf(tmpl, code)
}

// callGormMethod
type callGormMethod struct {
	gormMethodName string
	gormMethodArgs string
	gormVarName    string
}

func (m *callGormMethod) setGormMethodName(name string) {
	m.gormMethodName = name
}

func (m *callGormMethod) getGormMethodName() string {
	return m.gormMethodName
}

func (m *callGormMethod) getGormMethodArgs() string {
	return m.gormMethodArgs
}

func (m *callGormMethod) getGormVarName() string {
	return m.gormVarName
}

func (m callGormMethod) GetBody() string {
	return fmt.Sprintf("return %s.%s(%s)",
		m.getGormVarName(), m.getGormMethodName(), m.getGormMethodArgs())
}

func newCallGormMethod(name, args, varName string) callGormMethod {
	return callGormMethod{
		gormMethodName: name,
		gormMethodArgs: args,
		gormVarName:    varName,
	}
}

// dbArgMethod

type dbArgMethod struct {
	oneArgMethod
}

func newDbArgMethod() dbArgMethod {
	return dbArgMethod{
		oneArgMethod: newOneArgMethod("db", "*gorm.DB"),
	}
}

// gormErroredMethod
type gormErroredMethod struct {
	errorRetMethod
	callGormMethod
}

// GetBody returns body of method
func (m gormErroredMethod) GetBody() string {
	return m.callGormMethod.GetBody() + ".Error"
}

func newGormErroredMethod(name, args, varName string) gormErroredMethod {
	return gormErroredMethod{
		callGormMethod: newCallGormMethod(name, args, varName),
	}
}
