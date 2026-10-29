package gen

import (
	"fmt"
	"go/types"

	"github.com/borodyadka/inpupa/internal/annotation"
)

// kind — classification of a field type.
type kind int

const (
	kindInvalid kind = iota
	kindText         // a type with UnmarshalText
	kindBasic        // a basic kind or a named type over it
)

// fieldClass — the result of classifying a field type.
type fieldClass struct {
	kind      kind
	basic     *types.Basic // for kindBasic: the basic kind (unreliable if elem is named)
	isPtr     bool         // pointer layer: *T or *[]E
	isSlice   bool
	elem      types.Type     // T (for scalars/pointers), or E (slice element), or the mo.Option element E (option: true).
	option    bool           // a mo.Option[E] field (scalar sources only; body fields are never classified)
	optionPkg *types.Package // package of the Option origin (github.com/samber/mo) — for registering the mo import
}

// validField — a field with a valid annotation, ready for rendering.
type validField struct {
	fm  *fieldModel
	dir *annotation.Directive
	cls *fieldClass
}

// validStruct — a struct with one or more valid fields.
type validStruct struct {
	sm     *structModel
	fields []*validField
}

var errType = types.Universe.Lookup("error").Type()

// scalarKindsMsg — the scalar kinds list, verbatim (used as-is in the mo.Option element error:
// option element errors never reference Option support).
const scalarKindsMsg = "supported types: string; int, int8-int64; uint, uint8-uint64; float32, float64; bool; types with an UnmarshalText method"

// supportedTypesMsg — the top-level list: scalarKindsMsg plus exactly one trailing entry.
const supportedTypesMsg = scalarKindsMsg + "; mo.Option[E] where E is a supported scalar type"

// The backing option type: Option of module github.com/samber/mo.
const (
	moOptionPkgPath = "github.com/samber/mo"
	moOptionName    = "Option"
)

// validatePackage — validation of the whole package (E1–E10, W1).
func validatePackage(m *pkgModel, curPkg *types.Package) ([]Diagnostic, []*validStruct) {
	var diags []Diagnostic
	var valids []*validStruct

	for _, fm := range m.files {
		if !fm.isGen {
			diags = append(diags, scanOrphans(fm, m.fieldDocLn[fm.name])...)
		}
		for _, sm := range fm.structs {
			ds, vs := validateStruct(m, curPkg, sm)
			diags = append(diags, ds...)
			if len(vs.fields) > 0 {
				valids = append(valids, vs)
			}
		}
	}
	return diags, valids
}

func validateStruct(m *pkgModel, curPkg *types.Package, sm *structModel) ([]Diagnostic, *validStruct) {
	var diags []Diagnostic
	vs := &validStruct{sm: sm}

	type entry struct {
		fdm *fieldModel
		dir *annotation.Directive
		cls *fieldClass
	}
	clean := map[int]*entry{}
	firstByName := map[string]int{} // "source|name" → field index (scalars)
	bodyIdx := -1

	for i, fdm := range sm.fields {
		if len(fdm.anns) == 0 {
			continue
		}
		diags = append(diags, fieldDiags(sm, fdm)...)
		hasErr := len(fdm.anns) != 1 || fdm.anns[0].dir == nil || fdm.anns[0].perr != nil || fdm.goName == "" || !isExported(fdm.goName)
		if hasErr {
			continue
		}
		dir := fdm.anns[0].dir
		// Field-level E9, before any classification (body fields included).
		if p := typeParamRef(fdm.typ, sm.typeParams); p != "" {
			diags = append(diags, Diagnostic{
				Severity: SevError, Code: "E9",
				File: fdm.pos.Filename, Line: fdm.pos.Line, Col: fdm.pos.Column,
				Struct: sm.name, Field: fdm.goName, Directive: fdm.anns[0].line,
				Message: fmt.Sprintf("field %s uses type parameter %s; parsing into unbound type parameters is not supported", fdm.goName, p),
			})
			continue
		}
		var cls *fieldClass
		if dir.IsBody() {
			// body[json]: any type accepted by encoding/json is fine.
			cls = &fieldClass{elem: fdm.typ}
			if p, ok := fdm.typ.(*types.Pointer); ok {
				cls.isPtr = true
				cls.elem = p.Elem()
			}
		} else {
			var msg string
			cls, msg = classify(fdm.typ, dir.Source)
			if msg != "" {
				diags = append(diags, Diagnostic{
					Severity: SevError, Code: "E4",
					File: fdm.pos.Filename, Line: fdm.pos.Line, Col: fdm.pos.Column,
					Struct: sm.name, Field: fdm.goName, Directive: fdm.anns[0].line,
					Message: fmt.Sprintf("type %s: %s", typeString(fdm.typ, curPkg), msg),
				})
				continue
			}
		}
		if dir.IsBody() {
			if bodyIdx >= 0 {
				diags = append(diags, Diagnostic{
					Severity: SevError, Code: "E6",
					File: fdm.pos.Filename, Line: fdm.pos.Line, Col: fdm.pos.Column,
					Struct: sm.name, Field: fdm.goName, Directive: dir.Raw,
					Message: fmt.Sprintf("multiple body[...] fields in one struct (first: %s)", sm.fields[bodyIdx].goName),
				})
			} else {
				bodyIdx = i
			}
		} else {
			key := string(dir.Source) + "|" + dir.Name
			if j, ok := firstByName[key]; ok {
				diags = append(diags, Diagnostic{
					Severity: SevError, Code: "E5",
					File: fdm.pos.Filename, Line: fdm.pos.Line, Col: fdm.pos.Column,
					Struct: sm.name, Field: fdm.goName, Directive: dir.Raw,
					Message: fmt.Sprintf("wire name %q conflicts in source %s (already used by field %s)", dir.Name, dir.Source, sm.fields[j].goName),
				})
			} else {
				firstByName[key] = i
			}
		}
		clean[i] = &entry{fdm: fdm, dir: dir, cls: cls}
		vs.fields = append(vs.fields, &validField{fm: fdm, dir: dir, cls: cls})
	}

	if len(vs.fields) > 0 {
		if mp, ok := m.methods[sm.name]; ok {
			diags = append(diags, Diagnostic{
				Severity: SevError, Code: "E10",
				File: mp.Filename, Line: mp.Line, Col: mp.Column,
				Struct:  sm.name,
				Message: fmt.Sprintf("struct %s already has a user-defined FromRequest method in this package — conflicts with generation", sm.name),
			})
		}
	}
	return diags, vs
}

// fieldDiags — per-field errors E1/E2/E3/E7/E8.
func fieldDiags(sm *structModel, fdm *fieldModel) []Diagnostic {
	var out []Diagnostic
	for _, ann := range fdm.anns {
		if ann.perr != nil {
			out = append(out, Diagnostic{
				Severity: SevError, Code: ann.perr.Code,
				File: ann.pos.Filename, Line: ann.pos.Line, Col: ann.pos.Column,
				Struct: sm.name, Field: fdm.goName, Directive: ann.line,
				Message: ann.perr.Message, Hint: ann.perr.Hint,
			})
		}
	}
	if len(fdm.anns) >= 2 {
		out = append(out, Diagnostic{
			Severity: SevError, Code: "E3",
			File: fdm.anns[1].pos.Filename, Line: fdm.anns[1].pos.Line, Col: fdm.anns[1].pos.Column,
			Struct: sm.name, Field: fdm.goName, Directive: fdm.anns[1].line,
			Message: "two or more @input:* annotations on a single field",
			Hint:    "exactly one annotation per field",
		})
	}
	if fdm.goName == "" {
		out = append(out, Diagnostic{
			Severity: SevError, Code: "E8",
			File: fdm.pos.Filename, Line: fdm.pos.Line, Col: fdm.pos.Column,
			Struct: sm.name, Directive: firstAnnLine(fdm),
			Message: "annotation on an embedded (anonymous) field is not supported: the generator does not recurse through embeds",
			Hint:    "give the field a name and describe it explicitly",
		})
	} else if !isExported(fdm.goName) {
		out = append(out, Diagnostic{
			Severity: SevError, Code: "E7",
			File: fdm.pos.Filename, Line: fdm.pos.Line, Col: fdm.pos.Column,
			Struct: sm.name, Field: fdm.goName, Directive: firstAnnLine(fdm),
			Message: "annotation on an unexported field",
			Hint:    "the field must be exported (first letter uppercase)",
		})
	}
	return out
}

func firstAnnLine(fdm *fieldModel) string {
	if len(fdm.anns) == 0 {
		return ""
	}
	return fdm.anns[0].line
}

// isExported reports whether the name is exported.
func isExported(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)[0]
	return r >= 'A' && r <= 'Z'
}

// classify classifies a field type for the source.
func classify(t types.Type, src annotation.Source) (*fieldClass, string) {
	t = derefAlias(t)
	// The option rule on the declared type.
	if n, e, ok := optionField(t); ok {
		return classifyOption(e, n.Origin().Obj().Pkg(), src)
	}
	// Any other generic instantiation at field position.
	if n, ok := t.(*types.Named); ok && n.TypeArgs().Len() > 0 {
		return nil, "generic types are not supported, except mo.Option[E]"
	}
	isPtr := false
	if p, ok := t.(*types.Pointer); ok {
		isPtr = true
		t = p.Elem()
	}
	if s, ok := t.(*types.Slice); ok {
		if src != annotation.SourceQuery {
			return nil, fmt.Sprintf("slice fields are allowed for the query source only (source: %s)", src)
		}
		e := s.Elem()
		k, b, rej := classifyScalar(e)
		if k == kindInvalid {
			if rej != "" {
				return nil, rej
			}
			return nil, "the slice element type is not supported. " + supportedTypesMsg
		}
		return &fieldClass{kind: k, basic: b, isPtr: isPtr, isSlice: true, elem: e}, ""
	}
	k, b, rej := classifyScalar(t)
	if k == kindInvalid {
		if rej != "" {
			return nil, rej
		}
		return nil, "the field type is not supported for a scalar source. " + supportedTypesMsg
	}
	return &fieldClass{kind: k, basic: b, isPtr: isPtr, elem: t}, ""
}

// derefAlias resolves transparent aliases (*types.Alias) to their RHS types. Since Go 1.25 go/types makes
// transparent aliases first-class, an alias of an instantiation (type X = mo.Option[int]) is a *types.Alias,
// not the Named — Option matching must look through it. Defining aliases are ordinary Named types and
// pass through unchanged.
func derefAlias(t types.Type) types.Type {
	for {
		a, ok := t.(*types.Alias)
		if !ok {
			return t
		}
		t = a.Rhs()
	}
}

// optionField — mo.Option matching rule: the field type is a generic instantiation with exactly one
// type argument whose origin named type is Option of github.com/samber/mo (aliases resolve to the same Named).
func optionField(t types.Type) (*types.Named, types.Type, bool) {
	named, ok := t.(*types.Named)
	if !ok || named.TypeArgs().Len() != 1 {
		return nil, nil, false
	}
	obj := named.Origin().Obj()
	if obj.Pkg() == nil || obj.Pkg().Path() != moOptionPkgPath || obj.Name() != moOptionName {
		return nil, nil, false
	}
	return named, named.TypeArgs().At(0), true
}

// isMoOption — reports whether t (with at least one type argument) is an instantiation of github.com/samber/mo Option.
func isMoOption(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok || named.TypeArgs().Len() == 0 {
		return false
	}
	obj := named.Origin().Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == moOptionPkgPath && obj.Name() == moOptionName
}

// classifyOption classifies the element of a matched mo.Option field.
func classifyOption(e types.Type, pkg *types.Package, src annotation.Source) (*fieldClass, string) {
	e = derefAlias(e)
	if _, _, ok := optionField(e); ok {
		return nil, "nested mo.Option is not supported"
	}
	if _, ok := e.(*types.Pointer); ok {
		return nil, "pointer elements are not allowed inside mo.Option[E]"
	}
	if s, ok := e.(*types.Slice); ok {
		// mo.Option[[]E] — mirrors the plain []E rule: query only.
		if src != annotation.SourceQuery {
			return nil, fmt.Sprintf("slice fields are allowed for the query source only (source: %s)", src)
		}
		k, b, _ := classifyScalar(s.Elem())
		if k == kindInvalid {
			return nil, "the mo.Option element type is not supported. " + scalarKindsMsg
		}
		return &fieldClass{kind: k, basic: b, isSlice: true, elem: s.Elem(), option: true, optionPkg: pkg}, ""
	}
	k, b, _ := classifyScalar(e)
	if k == kindInvalid {
		return nil, "the mo.Option element type is not supported. " + scalarKindsMsg
	}
	return &fieldClass{kind: k, basic: b, elem: e, option: true, optionPkg: pkg}, ""
}

// classifyScalar classifies a scalar position.
// The third return value carries a dedicated rejection reason: a mo.Option origin reaching scalar
// classification (behind at most one pointer layer, e.g. *mo.Option[E] or an Option inside []E) is
// rejected with it instead of the list-based messages — checked before the TextUnmarshaler test because
// samber/mo declares UnmarshalText on *Option[T].
func classifyScalar(t types.Type) (kind, *types.Basic, string) {
	t = derefAlias(t)
	if moOptionInCompound(t) {
		return kindInvalid, nil, "mo.Option[E] is not supported inside pointers or slices"
	}
	if implementsTextUnmarshaler(t) {
		return kindText, nil, ""
	}
	if b, ok := t.Underlying().(*types.Basic); ok && supportedBasicKind(b.Kind()) {
		return kindBasic, b, ""
	}
	return kindInvalid, nil, ""
}

// moOptionInCompound — the compound-option test: unwraps at most one pointer layer before the
// Named check, so *mo.Option[E] (field position) and []*mo.Option[E] / []mo.Option[E] (element positions)
// all hit it.
func moOptionInCompound(t types.Type) bool {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	return isMoOption(t)
}

func supportedBasicKind(k types.BasicKind) bool {
	switch k {
	case types.String,
		types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
		types.Float32, types.Float64,
		types.Bool:
		return true
	default:
		return false
	}
}

// typeParamRef — the name of a struct type parameter referenced by t
// (recursive walk over pointer/slice/array elements, map keys/values and Named type arguments),
// or "" if none.
func typeParamRef(t types.Type, params []*types.TypeParam) string {
	if n := paramOf(t, params); n != "" {
		return n
	}
	switch tt := t.(type) {
	case *types.Pointer:
		return typeParamRef(tt.Elem(), params)
	case *types.Slice:
		return typeParamRef(tt.Elem(), params)
	case *types.Array:
		return typeParamRef(tt.Elem(), params)
	case *types.Map:
		if n := typeParamRef(tt.Key(), params); n != "" {
			return n
		}
		return typeParamRef(tt.Elem(), params)
	case *types.Named:
		for i := 0; i < tt.TypeArgs().Len(); i++ {
			if n := typeParamRef(tt.TypeArgs().At(i), params); n != "" {
				return n
			}
		}
	}
	return ""
}

func paramOf(t types.Type, params []*types.TypeParam) string {
	for _, p := range params {
		if types.Identical(t, p) {
			return p.Obj().Name()
		}
	}
	return ""
}

// implementsTextUnmarshaler reports whether *t implements encoding.TextUnmarshaler
// (method UnmarshalText([]byte) error; takes precedence over the basic kind).
func implementsTextUnmarshaler(t types.Type) bool {
	obj, _, _ := types.LookupFieldOrMethod(types.NewPointer(t), true, nil, "UnmarshalText")
	if obj == nil {
		return false
	}
	sig, ok := obj.Type().(*types.Signature)
	if !ok || sig.Params().Len() != 1 || sig.Results().Len() != 1 {
		return false
	}
	if !types.Identical(sig.Params().At(0).Type(), types.NewSlice(types.Typ[types.Uint8])) {
		return false
	}
	return types.Identical(sig.Results().At(0).Type(), errType)
}

// typeString — a type name for diagnostics (unqualified for the local package).
func typeString(t types.Type, curPkg *types.Package) string {
	return types.TypeString(t, func(p *types.Package) string {
		if p == nil || p == curPkg {
			return ""
		}
		return p.Name()
	})
}
