package gen

import (
	"errors"
	"go/ast"
	"go/scanner"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/borodyadka/inpupa/internal/annotation"
)

const loadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedTypes |
	packages.NeedTypesInfo |
	packages.NeedSyntax

// fileModel — a source file of the package (generated files are included and flagged with isGen).
type fileModel struct {
	name    string // base name of the file
	path    string
	isGen   bool
	ast     *ast.File
	structs []*structModel
}

// structModel — a named struct declaration.
type structModel struct {
	name       string
	pos        token.Position
	generic    bool
	typeParams []*types.TypeParam // type parameters in declaration order (nil for non-generic)
	fields     []*fieldModel
	file       *fileModel
}

// fieldAnn — an @input:* line in the doc group of a field.
type fieldAnn struct {
	line string // directive text (trimmed, without "//")
	dir  *annotation.Directive
	perr *annotation.Error
	pos  token.Position
}

// fieldModel — a struct field with extracted annotations.
type fieldModel struct {
	goName   string
	typ      types.Type
	pos      token.Position // position of the field name (or of the type for embedded)
	docStart int            // first line of the doc group; 0 if absent
	anns     []*fieldAnn
}

// pkgModel — a package model suitable for validation and rendering.
type pkgModel struct {
	name       string
	dir        string
	files      []*fileModel
	methods    map[string]token.Position // receiver name → position (user-defined FromRequest)
	fieldDocLn map[string]map[int]bool   // filename → lines that start a field doc group
}

// loadPackage loads and types the package in directory dir.
func loadPackage(dir string) (*packages.Package, *Diagnostic) {
	fset := token.NewFileSet()
	pkgs, err := packages.Load(&packages.Config{Mode: loadMode, Fset: fset, Dir: dir}, ".")
	if err != nil || len(pkgs) == 0 {
		msg := "failed to load the package"
		if err != nil {
			msg = err.Error()
		}
		return nil, &Diagnostic{Severity: SevError, Code: "LOAD", File: dir, Message: msg}
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		var msgs []string
		for _, pe := range pkg.Errors {
			msgs = append(msgs, pe.Msg)
		}
		return nil, &Diagnostic{Severity: SevError, Code: "LOAD", File: dir, Message: "package typing failed: " + strings.Join(msgs, "; ")}
	}
	return pkg, nil
}

// buildModel builds the package model.
func buildModel(pkg *packages.Package) *pkgModel {
	fset := pkg.Fset
	info := pkg.TypesInfo

	m := &pkgModel{
		name:       pkg.Name,
		dir:        pkg.Dir,
		methods:    map[string]token.Position{},
		fieldDocLn: map[string]map[int]bool{},
	}

	syntaxByFile := map[string]*ast.File{}
	for _, f := range pkg.Syntax {
		pos := fset.Position(f.Name.Pos())
		syntaxByFile[pos.Filename] = f
	}

	var paths []string
	for _, p := range pkg.CompiledGoFiles {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		f := syntaxByFile[p]
		if f == nil {
			continue
		}
		name := filepath.Base(p)
		fm := &fileModel{name: name, path: p, isGen: isGenName(name), ast: f}
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				sm := &structModel{name: ts.Name.Name, pos: fset.Position(ts.Name.Pos()), generic: ts.TypeParams != nil, file: fm}
				stAst, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				st, ok := asStructNamed(info.Defs[ts.Name])
				if !ok {
					continue // not a struct — no input
				}
				if tps := st.TypeParams(); tps != nil {
					for i := 0; i < tps.Len(); i++ {
						sm.typeParams = append(sm.typeParams, tps.At(i))
					}
				}
				su := st.Underlying().(*types.Struct)
				for i, field := range stAst.Fields.List {
					fm2 := &fieldModel{typ: su.Field(i).Type()}
					if len(field.Names) > 0 {
						fm2.goName = field.Names[0].Name
						fm2.pos = fset.Position(field.Names[0].Pos())
					} else {
						fm2.pos = fset.Position(field.Type.Pos())
					}
					fm2.anns = extractAnns(fset, field.Doc)
					if field.Doc != nil {
						fm2.docStart = fset.Position(field.Doc.Pos()).Line
					}
					sm.fields = append(sm.fields, fm2)
				}
				fm.structs = append(fm.structs, sm)
			}
		}
		m.files = append(m.files, fm)
	}

	// User-defined FromRequest methods (for E10).
	for _, fm := range m.files {
		if fm.isGen {
			continue
		}
		for _, decl := range fm.ast.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 || fd.Name.Name != "FromRequest" {
				continue
			}
			rf := fd.Recv.List[0]
			rt := rf.Type
			if se, ok := rt.(*ast.StarExpr); ok {
				rt = se.X
			}
			record := func(name string) {
				if _, exists := m.methods[name]; !exists {
					m.methods[name] = fset.Position(rf.Pos())
				}
			}
			switch base := rt.(type) {
			case *ast.Ident:
				record(base.Name)
			case *ast.IndexExpr: // a generic receiver, e.g. *Wrap[T]
				if ident, ok := base.X.(*ast.Ident); ok {
					record(ident.Name)
				}
			case *ast.IndexListExpr: // a generic receiver with several parameters
				if ident, ok := base.X.(*ast.Ident); ok {
					record(ident.Name)
				}
			}
		}
	}

	// First lines of field doc groups — for W1 (skip groups that belong to fields).
	for _, fm := range m.files {
		for _, sm := range fm.structs {
			for _, fdm := range sm.fields {
				if fdm.docStart > 0 {
					ensureMap(m.fieldDocLn, fm.name)[fdm.docStart] = true
				}
			}
		}
	}
	return m
}

func asStructNamed(obj types.Object) (*types.Named, bool) {
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return nil, false
	}
	named, ok := tn.Type().(*types.Named)
	if !ok {
		return nil, false
	}
	if _, ok = named.Underlying().(*types.Struct); !ok {
		return nil, false
	}
	return named, true
}

func ensureMap(m map[string]map[int]bool, k string) map[int]bool {
	if m[k] == nil {
		m[k] = map[int]bool{}
	}
	return m[k]
}

// extractAnns extracts @input:* lines from the doc group of a field.
func extractAnns(fset *token.FileSet, doc *ast.CommentGroup) []*fieldAnn {
	if doc == nil {
		return nil
	}
	var out []*fieldAnn
	for _, c := range doc.List {
		if !strings.HasPrefix(c.Text, "//") {
			continue
		}
		line := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		if !annotation.IsAnnotation(line) {
			continue
		}
		fa := &fieldAnn{line: line, pos: fset.Position(c.Pos())}
		d, err := annotation.Parse(line)
		fa.dir = d
		if err != nil {
			var pe *annotation.Error
			if errors.As(err, &pe) {
				fa.perr = pe
			}
		}
		out = append(out, fa)
	}
	return out
}

// scanOrphans finds @input:* lines in file comments that are not attached to any field (W1).
func scanOrphans(fm *fileModel, knownStartLines map[int]bool) []Diagnostic {
	src, err := os.ReadFile(fm.path)
	if err != nil {
		return nil
	}
	fset := token.NewFileSet()
	sf := fset.AddFile(fm.path, fset.Base(), len(src))
	var sc scanner.Scanner
	sc.Init(sf, src, nil, scanner.ScanComments)
	type cmt struct {
		line, col int
		text      string // trimmed text without "//"
	}
	var comments []cmt
	for {
		pos, tok, lit := sc.Scan()
		if tok == token.EOF {
			break
		}
		if tok != token.COMMENT {
			continue
		}
		if !strings.HasPrefix(lit, "//") {
			continue // line comments only
		}
		p := sf.Position(pos)
		comments = append(comments, cmt{line: p.Line, col: p.Column, text: strings.TrimSpace(strings.TrimPrefix(lit, "//"))})
	}
	type group struct {
		firstLine int
		anns      []cmt
	}
	var groups []group
	for i := 0; i < len(comments); {
		j := i
		for j+1 < len(comments) && comments[j+1].line == comments[j].line+1 {
			j++
		}
		g := group{firstLine: comments[i].line}
		for _, c := range comments[i : j+1] {
			if annotation.IsAnnotation(c.text) {
				g.anns = append(g.anns, c)
			}
		}
		if len(g.anns) > 0 {
			groups = append(groups, g)
		}
		i = j + 1
	}

	var diags []Diagnostic
	for _, g := range groups {
		if knownStartLines[g.firstLine] {
			continue // the group belongs to a field doc — handled above
		}
		for _, c := range g.anns {
			diags = append(diags, Diagnostic{
				Severity:  SevWarning,
				Code:      "W1",
				File:      fm.path,
				Line:      c.line,
				Col:       c.col,
				Directive: c.text,
				Message:   "annotation is not attached to a struct field and will be ignored",
			})
		}
	}
	return diags
}
