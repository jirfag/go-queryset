package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/tools/go/loader"
)

// StructField represents one field in struct
type StructField struct {
	name string            //
	typ  types.Type        // field/method/parameter type
	tag  reflect.StructTag // field tag; or nil
}

func (sf StructField) Name() string {
	return sf.name
}

func (sf StructField) Type() types.Type {
	return sf.typ
}

func (sf StructField) Tag() reflect.StructTag {
	return sf.tag
}

// ParsedStructs is a map from struct type name to list of fields
type ParsedStructs map[string]ParsedStruct

// ParsedStruct represents struct info
type ParsedStruct struct {
	TypeName string
	Fields   []StructField
	Doc      *ast.CommentGroup // line comments; or nil
}

func fileNameToPkgName(filePath, absFilePath string) string {
	dir := filepath.Dir(absFilePath)
	gopathEnv := os.Getenv("GOPATH")
	gopaths := strings.Split(gopathEnv, string(os.PathListSeparator))
	var inGoPath string
	for _, gopath := range gopaths {
		if strings.HasPrefix(dir, gopath) {
			inGoPath = gopath
			break
		}
	}
	if inGoPath == "" {
		// not in GOPATH
		return "./" + filepath.Dir(filePath)
	}
	r := strings.TrimPrefix(dir, inGoPath)
	r = strings.TrimPrefix(r, "/")  // may be and may not be
	r = strings.TrimPrefix(r, "\\") // may be and may not be
	r = strings.TrimPrefix(r, "src/")
	r = strings.TrimPrefix(r, "src\\")
	return r
}

func typeCheckFuncBodies(path string) bool {
	return false // don't type-check func bodies to speedup parsing
}

func loadProgramFromPackage(pkgFullName string) (*loader.Program, error) {
	// The loader loads a complete Go program from source code.
	conf := loader.Config{
		ParserMode:          parser.ParseComments,
		TypeCheckFuncBodies: typeCheckFuncBodies,
	}
	conf.Import(pkgFullName)
	lprog, err := conf.Load()
	if err != nil {
		return nil, fmt.Errorf("can't load program from package %q: %s",
			pkgFullName, err)
	}

	return lprog, nil
}

type structNamesInfo map[string]*ast.GenDecl

type structNamesVisitor struct {
	names      structNamesInfo
	curGenDecl *ast.GenDecl
}

func (v *structNamesVisitor) Visit(n ast.Node) (w ast.Visitor) {
	switch n := n.(type) {
	case *ast.GenDecl:
		v.curGenDecl = n
	case *ast.TypeSpec:
		if _, ok := n.Type.(*ast.StructType); ok {
			v.names[n.Name.Name] = v.curGenDecl
		}
	}

	return v
}

func getStructNamesInFile(fname string) (structNamesInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fname, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("can't parse file %q: %s", fname, err)
	}

	v := structNamesVisitor{
		names: structNamesInfo{},
	}
	ast.Walk(&v, f)
	return v.names, nil
}

// GetStructsInFile lists all structures in file passed and returns them with all fields
func GetStructsInFile(filePath string) (*loader.PackageInfo, ParsedStructs, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("can't get abs path for %s", filePath)
	}

	neededStructs, err := getStructNamesInFile(absFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("can't get struct names: %s", err)
	}

	packageFullName := fileNameToPkgName(filePath, absFilePath)
	lprog, err := loadProgramFromPackage(packageFullName)
	if err != nil {
		return nil, nil, err
	}

	pkgInfo := lprog.Package(packageFullName)
	if pkgInfo == nil {
		return nil, nil, fmt.Errorf("can't load types for file %s in package %q",
			filePath, packageFullName)
	}

	ret := ParsedStructs{}

	scope := pkgInfo.Pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)

		if neededStructs[name] == nil {
			continue
		}

		t := obj.Type().(*types.Named)
		s := t.Underlying().(*types.Struct)

		parsedStruct := parseStruct(s, neededStructs[name])
		if parsedStruct != nil {
			parsedStruct.TypeName = name
			ret[name] = *parsedStruct
		}
	}

	return pkgInfo, ret, nil
}

func newStructField(f *types.Var, tag string) *StructField {
	return &StructField{
		name: f.Name(),
		typ:  f.Type(),
		tag:  reflect.StructTag(tag),
	}
}

func parseStructFields(s *types.Struct) []StructField {
	var fields []StructField
	for i := 0; i < s.NumFields(); i++ {
		f := s.Field(i)
		if _, ok := f.Type().Underlying().(*types.Interface); ok {
			// skip interfaces
			continue
		}

		if f.Anonymous() {
			e, ok := f.Type().Underlying().(*types.Struct)
			if !ok {
				continue
			}

			pf := parseStructFields(e)
			if len(pf) == 0 {
				continue
			}

			fields = append(fields, pf...)
			continue
		}

		if !f.Exported() {
			continue
		}

		sf := newStructField(f, s.Tag(i))
		fields = append(fields, *sf)
	}

	return fields
}

func parseStruct(s *types.Struct, decl *ast.GenDecl) *ParsedStruct {
	fields := parseStructFields(s)
	if len(fields) == 0 {
		// e.g. no exported fields in struct
		return nil
	}

	var doc *ast.CommentGroup
	if decl != nil { // decl can be nil for embedded structs
		doc = decl.Doc // can obtain doc only from AST
	}

	return &ParsedStruct{
		Fields: fields,
		Doc:    doc,
	}
}
