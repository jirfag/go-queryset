package methods

import (
	"fmt"
	"strings"
)

func wrapToGormScope(code string) string {
	const tmpl = `return qs.w(%s)`
	return fmt.Sprintf(tmpl, code)
}

// methodCall
type methodCall struct {
	receiver   string
	methodName string
	methodArgs []interface{}
}

func (m *methodCall) setMethodName(name string) {
	m.methodName = name
}

func (m methodCall) getMethodName() string {
	return m.methodName
}

func (m *methodCall) setMethodArgs(args ...interface{}) {
	m.methodArgs = args
}

func (m methodCall) getReceiver() string {
	return m.receiver
}

func (m methodCall) GetBody() string {
	methodArgs := make([]string, len(m.methodArgs))
	for i, arg := range m.methodArgs {
		switch argType := arg.(type) {
		case string:
			methodArgs[i] = argType
		case methodCall:
			methodArgs[i] = argType.GetBody()
		default:
			panic(argType)
		}
	}
	return fmt.Sprintf("%s.%s(%s)",
		m.getReceiver(), m.getMethodName(), strings.Join(methodArgs, ", "))
}

func newMethodCall(receiver, methodName string, methodArgs ...interface{}) methodCall {
	return methodCall{
		receiver:   receiver,
		methodName: methodName,
		methodArgs: methodArgs,
	}
}

func newDBQuote(fieldName string) methodCall {
	return methodCall{
		receiver:   qsDbName,
		methodName: "Dialect().Quote",
		methodArgs: []interface{}{`"` + fieldName + `"`},
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
	methodCall
}

// GetBody returns body of method
func (m gormErroredMethod) GetBody() string {
	return "return " + m.methodCall.GetBody() + ".Error"
}

func newGormErroredMethod(name, args, varName string) gormErroredMethod {
	return gormErroredMethod{
		methodCall: newMethodCall(varName, name, args),
	}
}
