package methods

import "fmt"

func wrapToGormScope(code string) string {
	const tmpl = `qs.db = qs.db.Scopes(func(d *gorm.DB) *gorm.DB {
      %s})
    return qs`
	return fmt.Sprintf(tmpl, code)
}

// configurableGormMethod

type configurableGormMethod struct {
	gormMethodName string
}

func (m *configurableGormMethod) setGormMethodName(name string) {
	m.gormMethodName = name
}

func (m *configurableGormMethod) getGormMethodName() string {
	return m.gormMethodName
}

func newConfigurableGormMethod(name string) configurableGormMethod {
	return configurableGormMethod{gormMethodName: name}
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
