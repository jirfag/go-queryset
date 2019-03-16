package parser

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/errors"

	"golang.org/x/tools/go/packages"
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

// ParsedStruct represents struct info
type ParsedStruct struct {
	TypeName string
	Fields   []StructField
	Doc      *ast.CommentGroup // line comments; or nil
}

type Result struct {
	Structs     map[string]ParsedStruct
	PackageName string
	Types       *types.Package
}

type Structs struct{}

func (p Structs) ParseFile(ctx context.Context, filePath string) (*Result, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "can't get abs path for %s", filePath)
	}

	neededStructs, err := p.getStructNamesInFile(absFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "can't get struct names")
	}

	// need load the full package type info because
	// some deps can be in other files
	inPkgName := filepath.Dir(filePath)
	if !filepath.IsAbs(inPkgName) && !strings.HasPrefix(inPkgName, ".") {
		// to make this dir name a local package name
		// can't use filepath.Join because it calls Clean and removes "."+sep
		inPkgName = fmt.Sprintf(".%c%s", filepath.Separator, inPkgName)
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode:    packages.LoadAllSyntax,
		Context: ctx,
		Tests:   false,
	}, inPkgName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load package for file %s", filePath)
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf("got too many (%d) packages: %#v", len(pkgs), pkgs)
	}

	structs := p.buildParsedStructs(pkgs[0], neededStructs)
	return &Result{
		Structs:     structs,
		PackageName: pkgs[0].Name,
		Types:       pkgs[0].Types,
	}, nil
}

func (p Structs) buildParsedStructs(pkg *packages.Package, neededStructs structNamesInfo) map[string]ParsedStruct {
	ret := map[string]ParsedStruct{}

	scope := pkg.Types.Scope()
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
		} else {
			// TODO
		}
	}

	return ret
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

func (p Structs) getStructNamesInFile(fname string) (structNamesInfo, error) {
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
