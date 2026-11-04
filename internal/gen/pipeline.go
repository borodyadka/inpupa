package gen

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// generatePackage — validation and file writing for an already-loaded package.
func generatePackage(res *Result, opts Options, dir string, pkg *packages.Package, loadErr error) {
	if pkg == nil {
		msg := "failed to load the package"
		if loadErr != nil {
			msg = loadErr.Error()
		}
		res.Diagnostics = append(res.Diagnostics, Diagnostic{Severity: SevError, Code: "LOAD", File: dir, Message: msg})
		return
	}
	if len(pkg.Errors) > 0 {
		var msgs []string
		for _, pe := range pkg.Errors {
			msgs = append(msgs, pe.Msg)
		}
		res.Diagnostics = append(res.Diagnostics, Diagnostic{Severity: SevError, Code: "LOAD", File: dir, Message: "package typing failed: " + strings.Join(msgs, "; ")})
		return
	}
	m := buildModel(pkg)
	diags, valids := validatePackage(m, pkg.Types)
	res.Diagnostics = append(res.Diagnostics, diags...)
	if hasSev(diags, SevError) {
		logf(opts.Verbose, "  package %s: validation errors — files are not modified", pkg.PkgPath)
		return
	}

	byFile := map[*fileModel][]*validStruct{}
	for _, v := range valids {
		byFile[v.sm.file] = append(byFile[v.sm.file], v)
	}

	for _, fm := range m.files {
		if fm.isGen {
			continue
		}
		gp := genPathFor(fm.path)
		sts := byFile[fm]
		if len(sts) == 0 {
			if _, err := os.Stat(gp); err == nil {
				if err := os.Remove(gp); err == nil {
					res.Deleted = append(res.Deleted, gp)
					logf(opts.Verbose, "  removed stale %s", filepath.Base(gp))
				}
			}
			continue
		}
		r := newRenderer(pkg.Types, opts.Router)
		content := r.renderFile(pkg.Name, fm, sts)
		formatted, err := format.Source(content)
		if err != nil {
			res.Diagnostics = append(res.Diagnostics, Diagnostic{
				Severity: SevError, Code: "FMT", File: gp,
				Message: fmt.Sprintf("gofmt failed on generated code: %v%s", err, fmtContext(content, err)),
			})
			continue
		}
		if existing, e := os.ReadFile(gp); e == nil && string(existing) == string(formatted) {
			res.Unchanged = append(res.Unchanged, gp)
			logf(opts.Verbose, "  unchanged: %s", filepath.Base(gp))
			continue
		}
		if err := os.WriteFile(gp, formatted, 0o644); err != nil {
			res.Diagnostics = append(res.Diagnostics, Diagnostic{Severity: SevError, Code: "IO", File: gp, Message: err.Error()})
			continue
		}
		res.Written = append(res.Written, gp)
		logf(opts.Verbose, "  written: %s", filepath.Base(gp))
	}
}

func hasSev(ds []Diagnostic, s Severity) bool {
	for _, d := range ds {
		if d.Severity == s {
			return true
		}
	}
	return false
}

// fmtContext — lines around the gofmt error position (for diagnostics).
func fmtContext(src []byte, err error) string {
	var line int
	if _, e := fmt.Sscanf(err.Error(), "%d:", &line); e != nil || line <= 0 {
		return ""
	}
	lines := strings.Split(string(src), "\n")
	lo, hi := line-3, line+2
	if lo < 1 {
		lo = 1
	}
	if hi > len(lines) {
		hi = len(lines)
	}
	var b strings.Builder
	for i := lo; i <= hi; i++ {
		fmt.Fprintf(&b, "\n%4d: %s", i, lines[i-1])
	}
	return b.String()
}
