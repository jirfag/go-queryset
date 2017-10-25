package methods

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldNameToArgName(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, out string }{
		{"field", "field"},
		{"Field", "field"},
		{"MyField", "myField"},
		{"Type", "typeValue"}, // reserved keyword
		{"ID", "ID"},
		{"SOMENAME", "sOMENAME"}, // TODO
	}

	for _, c := range cases {
		assert.Equal(t, c.out, fieldNameToArgName(c.in))
	}
}
