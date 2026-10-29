package gen

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/borodyadka/inpupa/internal/annotation"
)

// runtimeImport — import of the inpupa runtime package in generated code.
const runtimeImport = "github.com/borodyadka/inpupa"

type imp struct{ path, name string }

type renderer struct {
	curPkg     *types.Package
	imports    map[string]string // path → import name (types from dependent packages)
	useHTTP    bool
	useStrconv bool
	useJSON    bool
	useErrors  bool
	useIO      bool
	useInpupa  bool
}

func newRenderer(curPkg *types.Package) *renderer {
	return &renderer{curPkg: curPkg, imports: map[string]string{}}
}

// tname — a type name for generated code.
func (r *renderer) tname(t types.Type) string {
	switch u := t.(type) {
	case *types.Named:
		p := u.Obj().Pkg()
		if p == nil || p == r.curPkg {
			return u.Obj().Name()
		}
		return r.importFor(p) + "." + u.Obj().Name()
	case *types.Basic:
		return u.Name()
	default:
		return types.TypeString(t, func(*types.Package) string { return "" })
	}
}

// importFor registers an import of the package and returns the name it is available under in the file.
func (r *renderer) importFor(p *types.Package) string {
	if n, ok := r.imports[p.Path()]; ok {
		return n
	}
	name := p.Name()
	i := 2
	for {
		conflict := false
		for usedName, path := range r.imports {
			if usedName == name && path != p.Path() {
				conflict = true
				break
			}
		}
		if !conflict {
			break
		}
		name = fmt.Sprintf("%s%d", p.Name(), i)
		i++
	}
	r.imports[p.Path()] = name
	return name
}

// stringVal — value expression for a string type (conversion for named types).
func (r *renderer) stringVal(t types.Type, expr string) string {
	if _, ok := t.(*types.Named); !ok {
		return expr
	}
	return r.tname(t) + "(" + expr + ")"
}

// scalarErr — return statement with a typed scalar parse error (raw is in scope).
func (r *renderer) scalarErr(S, F string, d *annotation.Directive) string {
	r.useInpupa = true
	return fmt.Sprintf("return &inpupa.Error{Kind: inpupa.KindScalar, Struct: %q, Field: %q, Source: %q, Name: %q, Value: raw, Err: err}", S, F, string(d.Source), d.Name)
}

// bodyErr — return statement with a body error (KindBody).
func (r *renderer) bodyErr(S, F string) string {
	r.useInpupa = true
	return fmt.Sprintf("return &inpupa.Error{Kind: inpupa.KindBody, Struct: %q, Field: %q, Source: %q, Err: err}", S, F, "body[json]")
}

// strconvCall — parsing expression for int/float/bool basic kinds.
func (r *renderer) strconvCall(b *types.Basic) string {
	r.useStrconv = true
	switch b.Kind() {
	case types.Int:
		return "strconv.ParseInt(raw, 10, strconv.IntSize)"
	case types.Int8:
		return "strconv.ParseInt(raw, 10, 8)"
	case types.Int16:
		return "strconv.ParseInt(raw, 10, 16)"
	case types.Int32:
		return "strconv.ParseInt(raw, 10, 32)"
	case types.Int64:
		return "strconv.ParseInt(raw, 10, 64)"
	case types.Uint:
		return "strconv.ParseUint(raw, 10, strconv.IntSize)"
	case types.Uint8:
		return "strconv.ParseUint(raw, 10, 8)"
	case types.Uint16:
		return "strconv.ParseUint(raw, 10, 16)"
	case types.Uint32:
		return "strconv.ParseUint(raw, 10, 32)"
	case types.Uint64:
		return "strconv.ParseUint(raw, 10, 64)"
	case types.Float32:
		return "strconv.ParseFloat(raw, 32)"
	case types.Float64:
		return "strconv.ParseFloat(raw, 64)"
	default:
		return "strconv.ParseBool(raw)"
	}
}

// valueParse — parses raw (in scope) into m.F; indent lvl.
func (r *renderer) valueParse(S, F string, d *annotation.Directive, cls *fieldClass, lvl int) []string {
	if cls.option {
		return r.optionValueParse(S, F, d, cls, lvl)
	}
	L := strings.Repeat("\t", lvl)
	var b []string
	if cls.kind == kindText {
		b = append(b,
			fmt.Sprintf("%sif err := m.%s.UnmarshalText([]byte(raw)); err != nil {", L, F),
			strings.Repeat("\t", lvl+1)+r.scalarErr(S, F, d),
			L+"}",
		)
		return b
	}
	if cls.basic.Kind() == types.String {
		b = append(b, fmt.Sprintf("%sm.%s = %s", L, F, r.stringVal(cls.elem, "raw")))
		return b
	}
	b = append(b,
		fmt.Sprintf("%spv, err := %s", L, r.strconvCall(cls.basic)),
		L+"if err != nil {",
		strings.Repeat("\t", lvl+1)+r.scalarErr(S, F, d),
		L+"}",
		fmt.Sprintf("%sm.%s = %s(pv)", L, F, r.tname(cls.elem)),
	)
	return b
}

// optionValueParse — the mo.Option variant of valueParse: the parsed value is wrapped in
// mo.Some; the assignment expression always carries the tname/stringVal conversion.
func (r *renderer) optionValueParse(S, F string, d *annotation.Directive, cls *fieldClass, lvl int) []string {
	L := strings.Repeat("\t", lvl)
	some := r.moSome(cls)
	if cls.kind == kindText {
		return []string{
			fmt.Sprintf("%svar v %s", L, r.tname(cls.elem)),
			fmt.Sprintf("%sif err := v.UnmarshalText([]byte(raw)); err != nil {", L),
			strings.Repeat("\t", lvl+1) + r.scalarErr(S, F, d),
			L + "}",
			fmt.Sprintf("%sm.%s = %s(v)", L, F, some),
		}
	}
	if cls.basic.Kind() == types.String {
		return []string{fmt.Sprintf("%sm.%s = %s(%s)", L, F, some, r.stringVal(cls.elem, "raw"))}
	}
	return []string{
		fmt.Sprintf("%spv, err := %s", L, r.strconvCall(cls.basic)),
		L + "if err != nil {",
		strings.Repeat("\t", lvl+1) + r.scalarErr(S, F, d),
		L + "}",
		fmt.Sprintf("%sm.%s = %s(%s(pv))", L, F, some, r.tname(cls.elem)),
	}
}

// moSome — the mo.Some constructor reference (registers the mo import via importFor).
func (r *renderer) moSome(cls *fieldClass) string {
	return r.importFor(cls.optionPkg) + ".Some"
}

// ptrParse — parses raw into a value of T and assigns the pointer m.F; indent lvl.
func (r *renderer) ptrParse(S, F string, d *annotation.Directive, cls *fieldClass, lvl int) []string {
	L := strings.Repeat("\t", lvl)
	var b []string
	if cls.kind == kindText {
		b = append(b,
			fmt.Sprintf("%svar v %s", L, r.tname(cls.elem)),
			fmt.Sprintf("%sif err := v.UnmarshalText([]byte(raw)); err != nil {", L),
			strings.Repeat("\t", lvl+1)+r.scalarErr(S, F, d),
			L+"}",
			fmt.Sprintf("%sm.%s = &v", L, F),
		)
		return b
	}
	if cls.basic.Kind() == types.String {
		b = append(b,
			fmt.Sprintf("%sv := %s", L, r.stringVal(cls.elem, "raw")),
			fmt.Sprintf("%sm.%s = &v", L, F),
		)
		return b
	}
	b = append(b,
		fmt.Sprintf("%spv, err := %s", L, r.strconvCall(cls.basic)),
		L+"if err != nil {",
		strings.Repeat("\t", lvl+1)+r.scalarErr(S, F, d),
		L+"}",
		fmt.Sprintf("%sv := %s(pv)", L, r.tname(cls.elem)),
		fmt.Sprintf("%sm.%s = &v", L, F),
	)
	return b
}

// elemParse — parses raw into v; returns code lines and the value expression.
func (r *renderer) elemParse(S, F string, d *annotation.Directive, cls *fieldClass, lvl int) ([]string, string) {
	L := strings.Repeat("\t", lvl)
	if cls.kind == kindText {
		return []string{
			fmt.Sprintf("%svar v %s", L, r.tname(cls.elem)),
			fmt.Sprintf("%sif err := v.UnmarshalText([]byte(raw)); err != nil {", L),
			strings.Repeat("\t", lvl+1) + r.scalarErr(S, F, d),
			L + "}",
		}, "v"
	}
	if cls.basic.Kind() == types.String {
		return nil, r.stringVal(cls.elem, "raw")
	}
	codes := []string{
		fmt.Sprintf("%spv, err := %s", L, r.strconvCall(cls.basic)),
		L + "if err != nil {",
		strings.Repeat("\t", lvl+1) + r.scalarErr(S, F, d),
		L + "}",
	}
	return codes, r.tname(cls.elem) + "(pv)"
}

// renderField — block that handles a single field (indent 1 = method body).
func (r *renderer) renderField(v *validStruct, vf *validField) []string {
	S := v.sm.name
	F := vf.fm.goName
	d := vf.dir
	cls := vf.cls

	switch {
	case d.IsBody():
		return r.renderBody(S, F, cls.isPtr)
	case cls.isSlice:
		if cls.option {
			// mo.Option[[]E] — query only (validated at classification).
			return r.querySliceOption(S, F, d, cls)
		}
		if cls.isPtr {
			return r.querySlicePtr(S, F, d, cls)
		}
		return r.querySlice(S, F, d, cls)
	case d.Source == annotation.SourceQuery:
		return r.fetchLoop(d.Name, S, F, d, cls)
	case d.Source == annotation.SourceHeader:
		return r.fetchGuard("req.Header.Get("+q(d.Name)+")", S, F, d, cls)
	case d.Source == annotation.SourcePath:
		return r.fetchGuard("req.PathValue("+q(d.Name)+")", S, F, d, cls)
	default: // cookie
		return r.renderCookie(S, F, d, cls)
	}
}

// fetchGuard — `if raw := <expr>; raw != "" { ... }` for header/path.
func (r *renderer) fetchGuard(expr, S, F string, d *annotation.Directive, cls *fieldClass) []string {
	var b []string
	b = append(b, fmt.Sprintf("\tif raw := %s; raw != \"\" {", expr))
	b = append(b, r.parseAt(S, F, d, cls, 2)...)
	b = append(b, "\t}")
	return b
}

// fetchLoop — loop over query values for a scalar field (first non-empty wins).
func (r *renderer) fetchLoop(name, S, F string, d *annotation.Directive, cls *fieldClass) []string {
	var b []string
	b = append(b, fmt.Sprintf("\tfor _, raw := range req.URL.Query()[%s] {", q(name)))
	b = append(b,
		"\t\tif raw == \"\" {",
		"\t\t\tcontinue",
		"\t\t}",
	)
	b = append(b, r.parseAt(S, F, d, cls, 2)...)
	b = append(b, "\t\tbreak", "\t}")
	return b
}

// parseAt — value/pointer parsing at the given indent.
func (r *renderer) parseAt(S, F string, d *annotation.Directive, cls *fieldClass, lvl int) []string {
	if cls.isPtr {
		return r.ptrParse(S, F, d, cls, lvl)
	}
	return r.valueParse(S, F, d, cls, lvl)
}

// querySlice — slice field (value) for query.
func (r *renderer) querySlice(S, F string, d *annotation.Directive, cls *fieldClass) []string {
	var b []string
	b = append(b, fmt.Sprintf("\tfor _, raw := range req.URL.Query()[%s] {", q(d.Name)))
	b = append(b,
		"\t\tif raw == \"\" {",
		"\t\t\tcontinue",
		"\t\t}",
	)
	codes, expr := r.elemParse(S, F, d, cls, 2)
	b = append(b, codes...)
	b = append(b, fmt.Sprintf("\t\tm.%s = append(m.%s, %s)", F, F, expr))
	b = append(b, "\t}")
	return b
}

// querySlicePtr — *[]E for query.
func (r *renderer) querySlicePtr(S, F string, d *annotation.Directive, cls *fieldClass) []string {
	var b []string
	b = append(b,
		fmt.Sprintf("\tif vals := req.URL.Query()[%s]; len(vals) > 0 {", q(d.Name)),
		fmt.Sprintf("\t\tvar list []%s", r.tname(cls.elem)),
		"\t\tfor _, raw := range vals {",
		"\t\t\tif raw == \"\" {",
		"\t\t\t\tcontinue",
		"\t\t\t}",
	)
	codes, expr := r.elemParse(S, F, d, cls, 3)
	b = append(b, codes...)
	b = append(b, fmt.Sprintf("\t\t\tlist = append(list, %s)", expr))
	b = append(b,
		"\t\t}",
		"\t\tif len(list) > 0 {",
		fmt.Sprintf("\t\t\tm.%s = &list", F),
		"\t\t}",
		"\t}",
	)
	return b
}

// querySliceOption — mo.Option[[]E] for query: accumulate all non-empty values and assign
// Some(list) only when at least one was parsed; otherwise the field stays None.
func (r *renderer) querySliceOption(S, F string, d *annotation.Directive, cls *fieldClass) []string {
	some := r.moSome(cls)
	var b []string
	b = append(b,
		fmt.Sprintf("\tif vals := req.URL.Query()[%s]; len(vals) > 0 {", q(d.Name)),
		fmt.Sprintf("\t\tvar list []%s", r.tname(cls.elem)),
		"\t\tfor _, raw := range vals {",
		"\t\t\tif raw == \"\" {",
		"\t\t\t\tcontinue",
		"\t\t\t}",
	)
	codes, expr := r.elemParse(S, F, d, cls, 3)
	b = append(b, codes...)
	b = append(b, fmt.Sprintf("\t\t\tlist = append(list, %s)", expr))
	b = append(b,
		"\t\t}",
		"\t\tif len(list) > 0 {",
		fmt.Sprintf("\t\t\tm.%s = %s(list)", F, some),
		"\t\t}",
		"\t}",
	)
	return b
}

// renderCookie — the cookie source.
func (r *renderer) renderCookie(S, F string, d *annotation.Directive, cls *fieldClass) []string {
	var b []string
	b = append(b,
		fmt.Sprintf("\tif c, cerr := req.Cookie(%s); cerr == nil {", q(d.Name)),
		"\t\tif raw := c.Value; raw != \"\" {",
	)
	b = append(b, r.parseAt(S, F, d, cls, 3)...)
	b = append(b, "\t\t}", "\t}")
	return b
}

// renderBody — body[json]. The decode target is always &m.F: for a value field this is *T,
// for a pointer field it is **T, so json allocates the value itself (empty body → nil without an error).
func (r *renderer) renderBody(S, F string, isPtr bool) []string {
	r.useJSON = true
	r.useErrors = true
	r.useIO = true
	var b []string
	b = append(b,
		fmt.Sprintf("\tif err := json.NewDecoder(req.Body).Decode(&m.%s); err != nil {", F),
	)
	if isPtr {
		b = append(b,
			"\t\tif !errors.Is(err, io.EOF) {",
			"\t\t\t"+r.bodyErr(S, F),
			"\t\t}",
			"\t}",
		)
		return b
	}
	b = append(b,
		"\t\tif errors.Is(err, io.EOF) {",
		fmt.Sprintf("\t\t\treturn &inpupa.Error{Kind: inpupa.KindBody, Struct: %q, Field: %q, Source: %q, Err: inpupa.ErrEmptyBody}", S, F, "body[json]"),
		"\t\t}",
		"\t\t"+r.bodyErr(S, F),
		"\t}",
	)
	return b
}

// receiverName — the method receiver type name; a generic struct repeats ALL its type parameter names in
// declaration order, no constraints (receivers take names only).
func receiverName(v *validStruct) string {
	name := v.sm.name
	if len(v.sm.typeParams) > 0 {
		names := make([]string, len(v.sm.typeParams))
		for i, tp := range v.sm.typeParams {
			names[i] = tp.Obj().Name()
		}
		name += "[" + strings.Join(names, ", ") + "]"
	}
	return name
}

// renderMethod — the whole FromRequest method (lines without a trailing \n).
func (r *renderer) renderMethod(v *validStruct) []string {
	var out []string
	out = append(out, fmt.Sprintf("func (m *%s) FromRequest(req *http.Request) error {", receiverName(v)))
	r.useHTTP = true
	out = append(out, "\tdefer req.Body.Close()")
	for _, vf := range v.fields {
		out = append(out, "")
		out = append(out, r.renderField(v, vf)...)
	}
	out = append(out, "", "\treturn nil", "}")
	return out
}

// renderFile — the whole file.
func (r *renderer) renderFile(pkgName string, fm *fileModel, structs []*validStruct) []byte {
	var body strings.Builder
	for _, v := range structs {
		body.WriteString("\n")
		for _, ln := range r.renderMethod(v) {
			body.WriteString(ln + "\n")
		}
	}
	var b strings.Builder
	b.WriteString("// Code generated by inpupa. DO NOT EDIT.\n")
	fmt.Fprintf(&b, "// Source: %s\n\n", fm.name)
	fmt.Fprintf(&b, "package %s\n", pkgName)
	if blk := r.importBlock(); blk != "" {
		b.WriteString(blk)
	}
	b.WriteString(body.String())
	return []byte(b.String())
}

// importBlock — the import block (a subset, sorted by path).
func (r *renderer) importBlock() string {
	var imps []imp
	if r.useHTTP {
		imps = append(imps, imp{"net/http", "http"})
	}
	if r.useStrconv {
		imps = append(imps, imp{"strconv", "strconv"})
	}
	if r.useJSON {
		imps = append(imps, imp{"encoding/json", "json"})
	}
	if r.useErrors {
		imps = append(imps, imp{"errors", "errors"})
	}
	if r.useIO {
		imps = append(imps, imp{"io", "io"})
	}
	if r.useInpupa {
		imps = append(imps, imp{runtimeImport, "inpupa"})
	}
	for path, name := range r.imports {
		imps = append(imps, imp{path, name})
	}
	sort.Slice(imps, func(i, j int) bool { return imps[i].path < imps[j].path })
	if len(imps) == 0 {
		return ""
	}
	var std, ext []imp
	for _, in := range imps {
		if isStdImport(in.path) {
			std = append(std, in)
		} else {
			ext = append(ext, in)
		}
	}
	var b strings.Builder
	b.WriteString("\nimport (\n")
	writeGroup(&b, std)
	if len(std) > 0 && len(ext) > 0 {
		b.WriteString("\n")
	}
	writeGroup(&b, ext)
	b.WriteString(")\n")
	return b.String()
}

func writeGroup(b *strings.Builder, imps []imp) {
	for _, in := range imps {
		fmt.Fprintf(b, "\t%s\n", q(in.path))
	}
}

// isStdImport — heuristic: a standard library package if the first path element has no dot.
func isStdImport(path string) bool {
	return !strings.Contains(strings.SplitN(path, "/", 2)[0], ".")
}

// q — quoting.
func q(s string) string { return fmt.Sprintf("%q", s) }
