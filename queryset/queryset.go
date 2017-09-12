package queryset

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/types"
	"io"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/tools/go/loader"

	"github.com/jinzhu/gorm"
	"github.com/jirfag/go-queryset/parser"
	"github.com/jirfag/go-queryset/queryset/methods"
)

var qsTmpl = template.Must(
	template.New("generator").
		Funcs(template.FuncMap{
			"lcf":      methods.LowercaseFirstRune,
			"todbname": gorm.ToDBName,
		}).
		Parse(qsCode),
)

type querySetStructConfig struct {
	StructName string
	Name       string
	Methods    methodsSlice
	Fields     []parser.StructField
}

type methodsSlice []methods.Method

func (s methodsSlice) Len() int { return len(s) }
func (s methodsSlice) Less(i, j int) bool {
	return strings.Compare(s[i].GetMethodName(), s[j].GetMethodName()) < 0
}
func (s methodsSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type querySetStructConfigSlice []querySetStructConfig

func (s querySetStructConfigSlice) Len() int { return len(s) }
func (s querySetStructConfigSlice) Less(i, j int) bool {
	return strings.Compare(s[i].Name, s[j].Name) < 0
}
func (s querySetStructConfigSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type baseFieldInfo struct {
	name      string // name of field
	typeName  string // name of type of field
	isStruct  bool
	isNumeric bool
	isTime    bool
}

type fieldInfo struct {
	pointed *baseFieldInfo
	baseFieldInfo
	isPointer bool
}

func (fi fieldInfo) getPointed() fieldInfo {
	return fieldInfo{
		baseFieldInfo: *fi.pointed,
	}
}

func getQuerySetMethodsForField(f fieldInfo, qsTypeName string) []methods.Method {
	fCtx := methods.NewQsFieldContext("", f.name, f.typeName, qsTypeName)
	basicTypeMethods := []methods.Method{
		methods.NewBinaryFilterMethod(fCtx.WithName("eq")),
		methods.NewBinaryFilterMethod(fCtx.WithName("ne")),
	}
	if !f.isTime {
		inMethod := methods.NewInFilterMethod(fCtx)
		notInMethod := methods.NewNotInFilterMethod(fCtx)
		basicTypeMethods = append(basicTypeMethods, inMethod, notInMethod)
	}

	numericMethods := []methods.Method{
		methods.NewBinaryFilterMethod(fCtx.WithName("lt")),
		methods.NewBinaryFilterMethod(fCtx.WithName("gt")),
		methods.NewBinaryFilterMethod(fCtx.WithName("lte")),
		methods.NewBinaryFilterMethod(fCtx.WithName("gte")),
		methods.NewOrderAscByMethod(f.name, qsTypeName),
		methods.NewOrderDescByMethod(f.name, qsTypeName),
	}

	if f.isNumeric {
		return append(basicTypeMethods, numericMethods...)
	}

	if f.isStruct {
		// Association was found (any struct or struct pointer)
		return []methods.Method{methods.NewPreloadMethod(f.name, qsTypeName)}
	}

	if f.isPointer {
		ptrMethods := getQuerySetMethodsForField(f.getPointed(), qsTypeName)
		return append(ptrMethods,
			methods.NewIsNullMethod(f.name, qsTypeName),
			methods.NewIsNotNullMethod(f.name, qsTypeName))
	}

	// it's a string
	return basicTypeMethods
}

func generateFieldInfo(pkgInfo *loader.PackageInfo, name string, typ fmt.Stringer, originalTypeName string) *fieldInfo {
	typeName := typ.String()
	if originalTypeName != "" {
		// it's needed to preserver typedef's original name
		typeName = originalTypeName
	}

	switch t := typ.(type) {
	case *types.Basic:
		return &fieldInfo{
			baseFieldInfo: baseFieldInfo{
				name:      name,
				typeName:  typeName,
				isNumeric: t.Info()&types.IsNumeric != 0,
			},
		}
	case *types.Named:
		otn := t.Obj().Name()
		if t.Obj().Pkg() != pkgInfo.Pkg {
			parts := strings.Split(typ.String(), "/")
			otn = parts[len(parts)-1]
		}
		return generateFieldInfo(pkgInfo, name, t.Underlying(), otn)
	case *types.Struct:
		if typeName == "time.Time" {
			return &fieldInfo{
				baseFieldInfo: baseFieldInfo{
					name:      name,
					typeName:  typeName,
					isNumeric: true,
					isTime:    true,
				},
			}
		}

		return &fieldInfo{
			baseFieldInfo: baseFieldInfo{
				name:     name,
				typeName: typeName,
				isStruct: true,
			},
		}
	case *types.Pointer:
		pf := generateFieldInfo(pkgInfo, name, t.Elem(), "")
		return &fieldInfo{
			baseFieldInfo: baseFieldInfo{
				name:     name,
				typeName: typeName,
			},
			isPointer: true,
			pointed:   &pf.baseFieldInfo,
		}
	default:
		// no filtering is needed
		return nil
	}
}

func getQuerySetFieldMethods(fields []fieldInfo, qsTypeName string) []methods.Method {
	ret := []methods.Method{}
	for _, f := range fields {
		methods := getQuerySetMethodsForField(f, qsTypeName)
		ret = append(ret, methods...)
	}

	return ret
}

func getUpdaterTypeName(structTypeName string) string {
	return structTypeName + "Updater"
}

func getUpdaterMethods(fields []fieldInfo, structTypeName string) []methods.Method {
	updaterTypeName := getUpdaterTypeName(structTypeName)
	ret := []methods.Method{methods.NewUpdaterUpdateMethod(updaterTypeName)}
	for _, f := range fields {
		if f.isPointer {
			// TODO
			continue
		}
		dbSchemaTypeName := structTypeName + "DBSchema"
		ret = append(ret,
			methods.NewUpdaterSetMethod(f.name, f.typeName, updaterTypeName,
				dbSchemaTypeName))
	}
	return ret
}

func getMethodsForStruct(structTypeName string, fieldInfos []fieldInfo) []methods.Method {
	qsTypeName := structTypeName + "QuerySet"

	ret := []methods.Method{
		methods.NewLimitMethod(qsTypeName),
		methods.NewAllMethod(structTypeName, qsTypeName),
		methods.NewOneMethod(structTypeName, qsTypeName),
		methods.NewGetUpdaterMethod(qsTypeName, getUpdaterTypeName(structTypeName)),
		methods.NewDeleteMethod(qsTypeName, structTypeName),
		methods.NewStructModifierMethod("Create", structTypeName),
		methods.NewStructModifierMethod("Delete", structTypeName),
	}

	fieldMethods := getQuerySetFieldMethods(fieldInfos, qsTypeName)
	ret = append(ret, fieldMethods...)

	ret = append(ret, getUpdaterMethods(fieldInfos, structTypeName)...)

	return ret
}

func doesNeedToGenerateQuerySet(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}

	for _, c := range doc.List {
		parts := strings.Split(strings.TrimSpace(c.Text), ":")
		ok := len(parts) == 2 &&
			strings.TrimSpace(strings.TrimPrefix(parts[0], "//")) == "gen" &&
			strings.TrimSpace(parts[1]) == "qs"
		if ok {
			return true
		}
	}

	return false
}

func generateQuerySetConfigs(pkgInfo *loader.PackageInfo,
	structs parser.ParsedStructs) querySetStructConfigSlice {

	querySetStructConfigs := querySetStructConfigSlice{}

	for structTypeName, ps := range structs {
		if !doesNeedToGenerateQuerySet(ps.Doc) {
			continue
		}

		fieldInfos := []fieldInfo{}
		for _, f := range ps.Fields {
			fi := generateFieldInfo(pkgInfo, f.Name, f.Type, "")
			if fi == nil {
				continue
			}
			fieldInfos = append(fieldInfos, *fi)
		}

		methods := getMethodsForStruct(structTypeName, fieldInfos)

		qsConfig := querySetStructConfig{
			StructName: structTypeName,
			Name:       structTypeName + "QuerySet",
			Methods:    methods,
			Fields:     ps.Fields,
		}
		sort.Sort(qsConfig.Methods)
		querySetStructConfigs = append(querySetStructConfigs, qsConfig)
	}

	return querySetStructConfigs
}

// GenerateQuerySetsForStructs is an internal method to retrieve querysets
// generated code from parsed structs
func GenerateQuerySetsForStructs(pkgInfo *loader.PackageInfo, structs parser.ParsedStructs) (io.Reader, error) {

	querySetStructConfigs := generateQuerySetConfigs(pkgInfo, structs)
	if len(querySetStructConfigs) == 0 {
		return nil, nil
	}

	sort.Sort(querySetStructConfigs)

	var b bytes.Buffer
	err := qsTmpl.Execute(&b, struct {
		Configs querySetStructConfigSlice
	}{
		Configs: querySetStructConfigs,
	})

	if err != nil {
		return nil, fmt.Errorf("can't generate structs query sets: %s", err)
	}

	return &b, nil
}

const qsCode = `
// ===== BEGIN of all query sets

{{ range .Configs }}
  // ===== BEGIN of query set {{ .Name }}

	// {{ .Name }} is an queryset type for {{ .StructName }}
  type {{ .Name }} struct {
	  db *gorm.DB
  }

  // New{{ .Name }} constructs new {{ .Name }}
  func New{{ .Name }}(db *gorm.DB) {{ .Name }} {
	  return {{ .Name }}{
		  db: db,
	  }
  }

	func (qs {{ .Name }}) w(db *gorm.DB) {{ .Name }} {
	  return New{{ .Name }}(db)
  }

	{{ range .Methods }}
		{{ .GetDoc .GetMethodName }}
		func ({{ .GetReceiverDeclaration }}) {{ .GetMethodName }}({{ .GetArgsDeclaration }})
		{{- .GetReturnValuesDeclaration }} {
      {{ .GetBody }}
		}
	{{ end }}

  // ===== END of query set {{ .Name }}

	// ===== BEGIN of {{ .StructName }} modifiers

	{{ $ft := printf "%s%s" .StructName "DBSchemaField" | lcf }}
	type {{ $ft }} string

	// {{ .StructName }}DBSchema stores db field names of {{ .StructName }}
	var {{ .StructName }}DBSchema = struct {
		{{ range .Fields }}
			{{ .Name }} {{ $ft }}
		{{- end }}
	}{
		{{ range .Fields }}
			{{ .Name }}: {{ $ft }}("{{ .Name | todbname }}"),
		{{- end }}
	}

	// Update updates {{ .StructName }} fields by primary key
	func (o *{{ .StructName }}) Update(db *gorm.DB, fields ...{{ $ft }}) error {
		dbNameToFieldName := map[string]interface{}{
			{{- range .Fields }}
				"{{ .Name | todbname }}": o.{{ .Name }},
			{{- end }}
		}
		u := map[string]interface{}{}
		for _, f := range fields {
			fs := string(f)
			u[fs] = dbNameToFieldName[fs]
		}
		if err := db.Model(o).Updates(u).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return err
			}

			return fmt.Errorf("can't update {{ .StructName }} %v fields %v: %s",
				o, fields, err)
		}

		return nil
	}

	// {{ .StructName }}Updater is an {{ .StructName }} updates manager
	type {{ .StructName }}Updater struct {
		fields map[string]interface{}
		db *gorm.DB
	}

	// New{{ .StructName }}Updater creates new {{ .StructName }} updater
	func New{{ .StructName }}Updater(db *gorm.DB) {{ .StructName }}Updater {
		return {{ .StructName }}Updater{
			fields: map[string]interface{}{},
			db: db.Model(&{{ .StructName }}{}),
		}
	}

	// ===== END of {{ .StructName }} modifiers
{{ end }}

// ===== END of all query sets
`
