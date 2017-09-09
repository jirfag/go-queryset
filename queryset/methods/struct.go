package methods

import "fmt"

// CreateMethod represents Create method
type CreateMethod struct {
	namedMethod
	structMethod
	dbArgMethod
	errorRetMethod
}

// GetBody returns body of Create method
func (m CreateMethod) GetBody() string {
	const tmpl = `if err := db.Create(o).Error; err != nil {
			return fmt.Errorf("can't create %s %%v: %%s", o, err)
		}
		return nil`
	return fmt.Sprintf(tmpl, m.structTypeName)
}

// NewCreateMethod create Create method
func NewCreateMethod(structTypeName string) CreateMethod {
	r := CreateMethod{
		namedMethod:  newNamedMethod("Create"),
		dbArgMethod:  newDbArgMethod(),
		structMethod: newStructMethod("o", "*"+structTypeName),
	}
	return r
}
