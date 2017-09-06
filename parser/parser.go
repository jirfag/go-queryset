package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/loader"
)

// StructField represents one field in struct
type StructField struct {
	Name    *ast.Ident         // field/method/parameter name; or nil if anonymous field
	Type    types.TypeAndValue // field/method/parameter type
	Tag     *ast.BasicLit      // field tag; or nil
	Comment *ast.CommentGroup  // line comments; or nil
}

// ParsedStructs is a map from struct type name to list of fields
type ParsedStructs map[ast.TypeSpec][]StructField

type visitor struct {
	structs       ParsedStructs
	neededStructs structNamesInfo
	typeInfo      types.Info

	currentStructType ast.TypeSpec
}

func (v *visitor) handleField(n *ast.Field) ast.Visitor {
	if len(n.Names) == 0 {
		id, ok := n.Type.(*ast.Ident)
		if !ok {
			// TODO: support embedded structs from another packages
			return nil
		}

		// embedded local struct, merge it into current struct
		var embedStructFields []StructField
		for t, f := range v.structs {
			if t.Name.Name == id.Name {
				embedStructFields = f
				break
			}
		}
		if embedStructFields == nil {
			log.Printf("can't find local embedded struct %s", id.Name)
			return nil
		}

		v.structs[v.currentStructType] = append(v.structs[v.currentStructType], embedStructFields...)
		return nil
	}

	// field found into structure
	for _, name := range n.Names {
		if !ast.IsExported(name.Name) {
			continue
		}

		t, ok := v.typeInfo.Types[n.Type]
		if !ok {
			log.Fatalf("no type for struct field %v", n)
		}

		sf := StructField{
			Name:    name,
			Type:    t,
			Tag:     n.Tag,
			Comment: n.Comment,
		}
		v.structs[v.currentStructType] = append(v.structs[v.currentStructType], sf)
	}

	return nil
}

// Visit method is needed to walk over AST tree
func (v *visitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.CommentGroup, *ast.Comment:
		return v
	case *ast.File:
		return v // go deeper
	case *ast.GenDecl:
		return v
	case *ast.TypeSpec:
		if _, ok := v.neededStructs[n.Name.Name]; !ok {
			return nil
		}
		// type decl found
		v.currentStructType = *n
		v.structs[v.currentStructType] = []StructField{}
		return v
	case *ast.StructType:
		// found type is a struct
		return v
	case *ast.FieldList:
		return v // go deeper
	case *ast.SelectorExpr:
		// TODO: need to save recursively walk into selected package
		// in order to support embedded structs from another packages
		return v
	case *ast.Field:
		return v.handleField(n)
	}

	return nil
}

func fileNameToPkgName(filePath string) string {
	return strings.TrimPrefix(filepath.Dir(filePath),
		fmt.Sprintf("%s/src/", os.Getenv("GOPATH")))
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

	if f.Name == nil {
		return nil, fmt.Errorf("package name wasn't found in file %q", fname)
	}

	v := structNamesVisitor{
		names: structNamesInfo{},
	}
	ast.Walk(&v, f)
	return v.names, nil
}

// GetStructsInFile lists all structures in file passed and returns them with all fields
func GetStructsInFile(fname string) (*loader.PackageInfo, ParsedStructs, error) {
	absFname, err := filepath.Abs(fname)
	if err != nil {
		return nil, nil, fmt.Errorf("can't get abs path for %s", fname)
	}

	neededStructs, err := getStructNamesInFile(absFname)
	if err != nil {
		return nil, nil, fmt.Errorf("can't get struct names: %s", err)
	}

	packageFullName := fileNameToPkgName(absFname)
	lprog, err := loadProgramFromPackage(packageFullName)
	if err != nil {
		return nil, nil, err
	}

	pkgInfo := lprog.Package(packageFullName)
	if pkgInfo == nil {
		return nil, nil, fmt.Errorf("can't load types for file %s in package %q",
			fname, packageFullName)
	}

	structs := getStructsFromPackage(pkgInfo, neededStructs)
	return pkgInfo, structs, nil
}

func getStructsFromPackage(pkgInfo *loader.PackageInfo, neededStructs structNamesInfo) ParsedStructs {

	v := visitor{
		structs:       ParsedStructs{},
		neededStructs: neededStructs,
		typeInfo:      pkgInfo.Info,
	}

	for _, f := range pkgInfo.Files {
		ast.Walk(&v, f)
	}

	for structName, structFields := range v.structs {
		if len(structFields) == 0 {
			// all the struct fields are unexported
			delete(v.structs, structName)
		}
	}

	// copy struct comments from ast.GenDecl to ast.TypeDecl
	for t, sf := range v.structs {
		delete(v.structs, t)
		t.Doc = neededStructs[t.Name.Name].Doc
		v.structs[t] = sf
	}

	return v.structs
}
