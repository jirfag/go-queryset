package methods

import (
	"github.com/zhaoshuyi-s0221/go-queryset/internal/parser"
	"github.com/zhaoshuyi-s0221/go-queryset/internal/queryset/field"
)

type QsStructContext struct {
	s parser.ParsedStruct
}

func NewQsStructContext(s parser.ParsedStruct) QsStructContext {
	return QsStructContext{
		s: s,
	}
}

func (ctx QsStructContext) qsTypeName() string {
	return ctx.s.TypeName + "QuerySet"
}

func (ctx QsStructContext) FieldCtx(f field.Info) QsFieldContext {
	return QsFieldContext{
		f:               f,
		QsStructContext: ctx,
	}
}

// QsFieldContext is a query set field context
type QsFieldContext struct {
	f             field.Info
	operationName string

	QsStructContext
}

func (ctx QsFieldContext) fieldName() string {
	return ctx.f.Name
}

func (ctx QsFieldContext) fieldDBName() string {
	return ctx.f.DBName
}

func (ctx QsFieldContext) fieldTypeName() string {
	return ctx.f.TypeName
}

func (ctx QsFieldContext) onFieldMethod() onFieldMethod {
	return newOnFieldMethod(ctx.operationName, ctx.fieldName())
}

func (ctx QsFieldContext) chainedQuerySetMethod() chainedQuerySetMethod {
	return newChainedQuerySetMethod(ctx.qsTypeName())
}

// WithOperationName return ctx with changed operation's name
func (ctx QsFieldContext) WithOperationName(operationName string) QsFieldContext {
	ctx.operationName = operationName
	return ctx
}
