package generator

import (
	"text/template"
)

var qsTmpl = template.Must(
	template.New("generator").
		Parse(qsCode),
)

const qsCode = `
// ===== BEGIN of all query sets

{{ range .Configs }}
  {{ $ft := printf "%s%s" .StructName "DBSchemaField" }}
  // ===== BEGIN of query set {{ .Name }}

	// {{ .Name }} is an queryset type for {{ .StructName }}
  type {{ .Name }} struct {
	  db *gorm.DB
  }

  // New{{ .Name }} constructs new {{ .Name }}
  func New{{ .Name }}(db *gorm.DB) {{ .Name }} {
	  return {{ .Name }}{
		  db: db.Model(&{{ .StructName }}{}),
	  }
  }

  func (qs {{ .Name }}) w(db *gorm.DB) {{ .Name }} {
	  return New{{ .Name }}(db)
  }

  func (qs {{ .Name }}) Select(fields ...{{ $ft }}) {{ .Name }} {
	  names := []string{}
	  for _, f := range fields {
		  names = append(names, f.String())
	  }

	  return qs.w(qs.db.Select(strings.Join(names, ",")))
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

	// {{ $ft }} describes database schema field. It requires for method 'Update'
	type {{ $ft }} string

	// String method returns string representation of field.
	// nolint: dupl
	func (f {{ $ft }}) String() string {
		return string(f)
	}

	// {{ .StructName }}DBSchema stores db field names of {{ .StructName }}
	var {{ .StructName }}DBSchema = struct {
		{{ range .Fields }}
			{{ .Name }} {{ $ft }}
		{{- end }}
	}{
		{{ range .Fields }}
			{{ .Name }}: {{ $ft }}("{{ .DBName }}"),
		{{- end }}
	}

	// Update updates {{ .StructName }} fields by primary key
	// nolint: dupl
	func (o *{{ .StructName }}) Update(db *gorm.DB, fields ...{{ $ft }}) error {
		dbNameToFieldName := map[string]interface{}{
			{{- range .Fields }}
				"{{ .DBName }}": o.{{ .Name }},
			{{- end }}
		}
		u := map[string]interface{}{}
		for _, f := range fields {
			fs := f.String()
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
	// nolint: dupl
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
