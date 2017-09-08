package queryset

import (
	"bytes"
	"fmt"
	"go/types"
	"io"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/tools/go/loader"

	"github.com/jinzhu/gorm"
	"github.com/jirfag/go-queryset/parser"
)

var qsTmpl = template.Must(
	template.New("generator").
		Funcs(template.FuncMap{
			"lcf":      lowercaseFirstRune,
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

type methodsSlice []method

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

func getMethodsForField(pkgInfo *loader.PackageInfo, name string, typ fmt.Stringer, originalTypeName string) []method {
	typeName := typ.String()
	if originalTypeName != "" {
		// it's needed to preserver typedef's original name
		typeName = originalTypeName
	}

	basicTypeMethods := []method{
		newBinaryFilterMethod("eq", name, typeName),
		newBinaryFilterMethod("ne", name, typeName),
	}
	numericMethods := []method{newBinaryFilterMethod("lt", name, typeName),
		newBinaryFilterMethod("gt", name, typeName),
		newBinaryFilterMethod("lte", name, typeName),
		newBinaryFilterMethod("gte", name, typeName),
		newOrderByMethod(name)}
	switch t := typ.(type) {
	case *types.Basic:
		if t.Info()&types.IsNumeric != 0 {
			return append(basicTypeMethods, numericMethods...)
		}
		// it's a string
		return basicTypeMethods
	case *types.Named:
		otn := t.Obj().Name()
		if t.Obj().Pkg() != pkgInfo.Pkg {
			if originalTypeName != "" {
				otn = fmt.Sprintf("%s.%s", t.Obj().Pkg().Name(), originalTypeName)
			} else {
				otn = typ.String()
			}
		}
		return getMethodsForField(pkgInfo, name, t.Underlying(), otn)
	case *types.Struct:
		if originalTypeName == "time.Time" {
			return append(basicTypeMethods, numericMethods...)
		}

		// Association was found (any struct or struct pointer)
		return []method{newPreloadMethod(name)}
	case *types.Pointer:
		ptrMethods := getMethodsForField(pkgInfo, name, t.Elem(), "")
		return append(ptrMethods, newIsNullMethod(name))
	default:
		// no filtering is needed
		return nil
	}
}

func getQuerySetFieldMethods(pkgInfo *loader.PackageInfo, fields []parser.StructField) []method {
	ret := []method{}
	for _, f := range fields {
		methods := getMethodsForField(pkgInfo, f.Name, f.Type, "")
		ret = append(ret, methods...)
	}

	return ret
}

// GenerateQuerySetsForStructs is an internal method to retrieve querysets
// generated code from parsed structs
func GenerateQuerySetsForStructs(pkgInfo *loader.PackageInfo, structs parser.ParsedStructs) (io.Reader, error) {
	querySetStructConfigs := querySetStructConfigSlice{}

	for structTypeName, ps := range structs {
		doc := ps.Doc
		if doc == nil {
			continue
		}

		ok := false
		for _, c := range doc.List {
			parts := strings.Split(strings.TrimSpace(c.Text), ":")
			ok = len(parts) == 2 &&
				strings.TrimSpace(strings.TrimPrefix(parts[0], "//")) == "gen" &&
				strings.TrimSpace(parts[1]) == "qs"
			if ok {
				break
			}
		}
		if !ok {
			continue
		}

		methods := []method{newLimitMethod(), newAllMethod(structTypeName),
			newOneMethod(structTypeName)}
		fieldMethods := getQuerySetFieldMethods(pkgInfo, ps.Fields)
		methods = append(methods, fieldMethods...)

		qsConfig := querySetStructConfig{
			StructName: structTypeName,
			Name:       structTypeName + "QuerySet",
			Methods:    methods,
			Fields:     ps.Fields,
		}
		sort.Sort(qsConfig.Methods)
		querySetStructConfigs = append(querySetStructConfigs, qsConfig)
	}

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

  {{ $qSTypeName := .Name }}

	{{ range .Methods }}
		{{ .GetDoc .GetMethodName }}
		func (qs {{ $qSTypeName }}) {{ .GetMethodName }}({{ .GetArgsDeclaration }})
		{{- .GetReturnValuesDeclaration $qSTypeName }} {
      {{ .GetBody }}
		}
	{{ end }}

  // ===== END of query set {{ .Name }}

	// ===== BEGIN of {{ .StructName }} modifiers

	// Create creates {{ .StructName }}
	func (o *{{ .StructName }}) Create(db *gorm.DB) error {
		if err := db.Create(o).Error; err != nil {
			return fmt.Errorf("can't create {{ .StructName }} %v: %s", o, err)
		}

		return nil
	}

	{{ $ft := printf "%s%s" .StructName "DBSchemaField" | lcf }}
	type {{ $ft }} string

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

	// ===== END of {{ .StructName }} modifiers
{{ end }}

// ===== END of all query sets
`
