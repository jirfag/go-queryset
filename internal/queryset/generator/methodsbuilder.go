package generator

import (
	"github.com/zhaoshuyi-s0221/go-queryset/internal/parser"
	"github.com/zhaoshuyi-s0221/go-queryset/internal/queryset/field"
	"github.com/zhaoshuyi-s0221/go-queryset/internal/queryset/methods"
)

type methodsBuilder struct {
	fields []field.Info
	s      parser.ParsedStruct
	ret    []methods.Method
	sctx   methods.QsStructContext
}

func (b *methodsBuilder) qsTypeName() string {
	return b.s.TypeName + "QuerySet"
}

func newMethodsBuilder(s parser.ParsedStruct, fields []field.Info) *methodsBuilder {
	return &methodsBuilder{
		s:      s,
		sctx:   methods.NewQsStructContext(s),
		fields: fields,
	}
}

func (b *methodsBuilder) getQuerySetMethodsForField(f field.Info) []methods.Method {
	fctx := b.sctx.FieldCtx(f)
	basicTypeMethods := []methods.Method{
		methods.NewBinaryFilterMethod(fctx.WithOperationName("eq")),
		methods.NewBinaryFilterMethod(fctx.WithOperationName("ne")),
		methods.NewOrderAscByMethod(fctx),
		methods.NewOrderDescByMethod(fctx),
	}

	if !f.IsTime {
		inMethod := methods.NewInFilterMethod(fctx)
		notInMethod := methods.NewNotInFilterMethod(fctx)
		basicTypeMethods = append(basicTypeMethods, inMethod, notInMethod)
	}

	numericMethods := []methods.Method{
		methods.NewBinaryFilterMethod(fctx.WithOperationName("lt")),
		methods.NewBinaryFilterMethod(fctx.WithOperationName("gt")),
		methods.NewBinaryFilterMethod(fctx.WithOperationName("lte")),
		methods.NewBinaryFilterMethod(fctx.WithOperationName("gte")),
	}

	if f.IsString {
		likeMethod := methods.NewBinaryFilterMethod(fctx.WithOperationName("like"))
		notLikeMethod := methods.NewBinaryFilterMethod(fctx.WithOperationName("notlike"))

		methods := append(basicTypeMethods, likeMethod, notLikeMethod)
		return append(methods, numericMethods...)
	}

	if f.IsNumeric {
		return append(basicTypeMethods, numericMethods...)
	}

	if f.IsStruct {
		// Association was found (any struct or struct pointer)
		return []methods.Method{methods.NewPreloadMethod(fctx)}
	}

	if f.IsPointer {
		ptrMethods := b.getQuerySetMethodsForField(f.GetPointed())
		return append(ptrMethods,
			methods.NewIsNullMethod(fctx),
			methods.NewIsNotNullMethod(fctx))
	}

	// it's a string
	return basicTypeMethods
}

func (b *methodsBuilder) buildQuerySetFieldMethods(f field.Info) *methodsBuilder {
	methods := b.getQuerySetMethodsForField(f)
	b.ret = append(b.ret, methods...)
	return b
}

func getUpdaterTypeName(structTypeName string) string {
	return structTypeName + "Updater"
}

func (b *methodsBuilder) buildUpdaterStructMethods() {
	updaterTypeName := getUpdaterTypeName(b.s.TypeName)
	b.ret = append(b.ret,
		methods.NewUpdaterUpdateMethod(updaterTypeName),
		methods.NewUpdaterUpdateNumMethod(updaterTypeName),
	)
}

func (b *methodsBuilder) buildUpdaterFieldMethods(f field.Info) {
	if f.IsPointer {
		p := f.GetPointed()
		if p.IsStruct {
			// TODO
			return
		}

		// It's a pointer to simple field (string, int).
		// Developer used pointer to distinguish between NULL and not NULL values.
	}

	dbSchemaTypeName := b.s.TypeName + "DBSchema"
	updaterTypeName := getUpdaterTypeName(b.s.TypeName)
	b.ret = append(b.ret,
		methods.NewUpdaterSetMethod(f.Name, f.TypeName, updaterTypeName,
			dbSchemaTypeName))
}

func (b *methodsBuilder) buildStructSelectMethods() *methodsBuilder {
	b.ret = append(b.ret,
		methods.NewAllMethod(b.s.TypeName, b.qsTypeName()),
		methods.NewOneMethod(b.s.TypeName, b.qsTypeName()),
		methods.NewLimitMethod(b.qsTypeName()),
		methods.NewOffsetMethod(b.qsTypeName()))
	return b
}

func (b *methodsBuilder) buildAggrMethods() *methodsBuilder {
	b.ret = append(b.ret,
		methods.NewCountMethod(b.qsTypeName()))
	return b
}

func (b *methodsBuilder) buildCRUDMethods() *methodsBuilder {
	b.ret = append(b.ret,
		methods.NewGetUpdaterMethod(b.qsTypeName(), getUpdaterTypeName(b.s.TypeName)),
		methods.NewDeleteMethod(b.qsTypeName(), b.s.TypeName),
		methods.NewStructModifierMethod("Create", b.s.TypeName),
		methods.NewStructModifierMethod("Delete", b.s.TypeName),
		methods.NewDeleteNumMethod(b.qsTypeName(), b.s.TypeName),
		methods.NewDeleteNumUnscopedMethod(b.qsTypeName(), b.s.TypeName),
		methods.NewGetDBMethod(b.qsTypeName()),
	)

	return b
}

func (b methodsBuilder) Build() []methods.Method {
	b.buildStructSelectMethods().
		buildAggrMethods().
		buildCRUDMethods().
		buildUpdaterStructMethods()

	for _, f := range b.fields {
		b.buildQuerySetFieldMethods(f).buildUpdaterFieldMethods(f)
	}

	return b.ret
}
