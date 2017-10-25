package queryset

import (
	"github.com/jirfag/go-queryset/parser"
	"github.com/jirfag/go-queryset/queryset/field"
	"github.com/jirfag/go-queryset/queryset/methods"
	"golang.org/x/tools/go/loader"
)

type methodsBuilder struct {
	pkgInfo *loader.PackageInfo
	s       parser.ParsedStruct
	ret     []methods.Method
}

func (b *methodsBuilder) qsTypeName() string {
	return b.s.TypeName + "QuerySet"
}

func newMethodsBuilder(pkgInfo *loader.PackageInfo, s parser.ParsedStruct) *methodsBuilder {
	return &methodsBuilder{
		pkgInfo: pkgInfo,
		s:       s,
	}
}

func getQuerySetMethodsForField(f field.Info, qsTypeName string) []methods.Method {
	fCtx := methods.NewQsFieldContext("", f.Name, f.TypeName, qsTypeName)
	basicTypeMethods := []methods.Method{
		methods.NewBinaryFilterMethod(fCtx.WithName("eq")),
		methods.NewBinaryFilterMethod(fCtx.WithName("ne")),
	}
	if !f.IsTime {
		inMethod := methods.NewInFilterMethod(fCtx)
		notInMethod := methods.NewNotInFilterMethod(fCtx)
		basicTypeMethods = append(basicTypeMethods, inMethod, notInMethod)
	}

	numericMethods := []methods.Method{
		methods.NewBinaryFilterMethod(fCtx.WithName("lt")),
		methods.NewBinaryFilterMethod(fCtx.WithName("gt")),
		methods.NewBinaryFilterMethod(fCtx.WithName("lte")),
		methods.NewBinaryFilterMethod(fCtx.WithName("gte")),
		methods.NewOrderAscByMethod(f.Name, qsTypeName),
		methods.NewOrderDescByMethod(f.Name, qsTypeName),
	}

	if f.IsNumeric {
		return append(basicTypeMethods, numericMethods...)
	}

	if f.IsStruct {
		// Association was found (any struct or struct pointer)
		return []methods.Method{methods.NewPreloadMethod(f.Name, qsTypeName)}
	}

	if f.IsPointer {
		ptrMethods := getQuerySetMethodsForField(f.GetPointed(), qsTypeName)
		return append(ptrMethods,
			methods.NewIsNullMethod(f.Name, qsTypeName),
			methods.NewIsNotNullMethod(f.Name, qsTypeName))
	}

	// it's a string
	return basicTypeMethods
}

func (b *methodsBuilder) buildQuerySetFieldMethods(f field.Info) *methodsBuilder {
	methods := getQuerySetMethodsForField(f, b.qsTypeName())
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
		methods.NewLimitMethod(b.qsTypeName()))
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
		methods.NewStructModifierMethod("Delete", b.s.TypeName))
	return b
}

func (b methodsBuilder) Build() []methods.Method {
	b.buildStructSelectMethods().
		buildAggrMethods().
		buildCRUDMethods().
		buildUpdaterStructMethods()

	g := field.NewInfoGenerator(b.pkgInfo)
	for _, f := range b.s.Fields {
		fi := g.GenFieldInfo(f)
		if fi == nil {
			continue
		}

		b.buildQuerySetFieldMethods(*fi).buildUpdaterFieldMethods(*fi)
	}

	return b.ret
}
