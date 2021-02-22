package parser

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getRepoRoot() string {
	_, selfFilePath, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("can't get caller")
	}

	root, err := filepath.Abs(filepath.Join(filepath.Dir(selfFilePath), ".."))
	if err != nil {
		log.Fatalf("can't get repo root: %s", err)
	}

	return root
}

func getTempDirRoot() string {
	return filepath.Join(getRepoRoot(), "parser", "test")
}

func getTempFileName(rootDir, prefix, suffix string) (*os.File, error) {
	randBytes := make([]byte, 16)
	_, err := rand.Read(randBytes)
	if err != nil {
		return nil, fmt.Errorf("can't generate random bytes: %s", err)
	}

	p := filepath.Join(rootDir, prefix+hex.EncodeToString(randBytes)+suffix)
	return os.Create(p)
}

func getTmpFileForCode(code string) *os.File {
	tmpDir, err := ioutil.TempDir(getTempDirRoot(), "tmptestdir")
	if err != nil {
		log.Fatalf("can't create temp dir: %s", err)
	}

	f, err := getTempFileName(tmpDir, "go-queryset-test", ".go")
	if err != nil {
		log.Fatalf("can't create temp file: %s", err)
	}

	_, err = f.Write([]byte(code))
	if err != nil {
		log.Fatalf("can't write to temp file %q: %s", f.Name(), err)
	}

	return f
}

func removeTempFileAndDir(f *os.File) {
	root := filepath.Dir(f.Name())
	if err := os.RemoveAll(root); err != nil {
		log.Fatalf("can't remove files from root %s: %s", root, err)
	}
}

func TestGetStructNamesInFile(t *testing.T) {
	cases := []struct {
		code                string
		expectedStructNames []string
		errorIsExpected     bool
	}{
		{
			code:            "",
			errorIsExpected: true,
		},
		{
			code: `package p
				type T struct {}`,
			expectedStructNames: []string{"T"},
		},
		{
			code: `package p
				type T1 struct {}
				type T2 struct {}`,
			expectedStructNames: []string{"T1", "T2"},
		},
		{
			code: `package p
				type T1 int
				type T2 struct {}`,
			expectedStructNames: []string{"T2"},
		},
		{
			code: `package p
				var v struct {F int}`,
		},
		{
			code: `package p
				const c = 1`,
		},
	}

	p := Structs{}
	for i, tc := range cases {
		tc := tc // capture range variable
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			t.Parallel()
			f := getTmpFileForCode(tc.code)
			defer removeTempFileAndDir(f)

			res, err := p.getStructNamesInFile(f.Name())
			if tc.errorIsExpected {
				assert.NotNil(t, err)
				return
			}

			assert.Nil(t, err)
			if tc.expectedStructNames == nil {
				tc.expectedStructNames = []string{}
			}
			for _, expStructName := range tc.expectedStructNames {
				decl, ok := res[expStructName]
				assert.True(t, ok, "no struct %s", expStructName)
				assert.NotNil(t, decl)
			}
			assert.Len(t, res, len(tc.expectedStructNames))
		})
	}
}

type structFieldsCase struct {
	code                 string
	expectedStructFields []string
	errorIsExpected      bool
	expectedDoc          []string
	expectedStructsCount int
}

func (tc structFieldsCase) getExpectedtructsCount() int {
	expectedStructsCount := 1
	if tc.expectedStructFields == nil {
		expectedStructsCount = 0
	}
	if tc.expectedStructsCount != 0 {
		expectedStructsCount = tc.expectedStructsCount
	}

	return expectedStructsCount
}

func TestGetStructsInFile(t *testing.T) {
	cases := []structFieldsCase{
		{
			code:            "",
			errorIsExpected: true,
		},
		{
			code: `package p
				type T struct {}`,
		},
		{
			code: `package p
				type T struct {
					F int
				}`,
			expectedStructFields: []string{"F"},
		},
		{
			code: `package p
				type T struct {
					f int
				}`,
		},
		{
			code: `package p
				// doc line 1
				// doc line 2
				type T struct {
					F int
				}`,
			expectedStructFields: []string{"F"},
			expectedDoc:          []string{"// doc line 1", "// doc line 2"},
		},
		{
			code: `package p
				type m struct {
					ID int
				}

				type T struct {
					m
					F int
				}`,
			expectedStructFields: []string{"F", "ID"},
			expectedStructsCount: 2,
		},
		{ // test local reordered embedding
			code: `package p
				type T struct {
					m
					F int
				}
				type m struct {
					ID int
				}`,
			expectedStructFields: []string{"F", "ID"},
			expectedStructsCount: 2,
		},
		{ // test another package imported embedding
			code: `package p
				import "gorm.io/gorm"
				type T struct {
					gorm.Model
					F int
				}`,
			expectedStructFields: []string{"ID", "CreatedAt", "UpdatedAt", "DeletedAt", "F"},
		},
		{
			code: `package p
			type MyType int`,
		},
		{
			code: `package p
				type m struct {
				}

				type T struct {
					m
					F int
				}`,
			expectedStructFields: []string{"F"},
		},
	}

	for i, tc := range cases {
		tc := tc // capture range variable
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			t.Parallel()
			testStructFields(t, tc)
		})
	}
}

func testStructFields(t *testing.T, tc structFieldsCase) {
	f := getTmpFileForCode(tc.code)
	defer removeTempFileAndDir(f)

	p := Structs{}
	ret, err := p.ParseFile(context.Background(), f.Name())
	if tc.errorIsExpected {
		assert.NotNil(t, err)
		return
	}

	assert.Nil(t, err)
	assert.NotNil(t, ret)

	assert.Len(t, ret.Structs, tc.getExpectedtructsCount())
	if tc.getExpectedtructsCount() == 0 {
		return
	}

	var typeName string

	for structTypeName := range ret.Structs {
		if structTypeName == "T" {
			typeName = structTypeName
			break
		}
	}
	assert.NotNil(t, typeName)

	s := ret.Structs[typeName]
	fieldNames := []string{}
	for _, field := range s.Fields {
		assert.NotEmpty(t, field.Name)
		fieldNames = append(fieldNames, field.name)
	}
	assert.Len(t, fieldNames, len(tc.expectedStructFields))

	if tc.expectedDoc != nil {
		docLines := []string{}
		assert.NotNil(t, s.Doc)
		for _, docLine := range s.Doc.List {
			docLines = append(docLines, docLine.Text)
		}
		assert.Equal(t, tc.expectedDoc, docLines)
	}
}
