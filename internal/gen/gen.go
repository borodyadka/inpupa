// Package gen implements generation of the FromRequest method from @input:* annotations.
//
// Pipeline: discover packages under the input directory → load types via go/packages →
// validate E1–E10/W1 → render <basename>.inpupa.gen.go next to each file with annotated structs.
package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GenFileSuffix — suffix of generated files; kept unique so the generator does not collide with others.
const GenFileSuffix = ".inpupa.gen.go"

// Options — generation parameters.
type Options struct {
	InputDir string   // root directory to scan
	Exclude  []string // directories relative to InputDir, skipped entirely
	Verbose  bool     // verbose logging on stderr
}

// Severity — diagnostic severity level.
type Severity string

const (
	SevError   Severity = "error"
	SevWarning Severity = "warning"
)

// Diagnostic — a single unit of generator output.
type Diagnostic struct {
	Severity  Severity
	Code      string // E1..E10, W1, LOAD, FMT
	File      string
	Line      int
	Col       int
	Struct    string
	Field     string
	Directive string // original @input:* line, if any
	Message   string
	Hint      string
}

// Render formats the diagnostic into a single line.
func (d Diagnostic) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] ", d.Severity)
	if d.File != "" {
		fmt.Fprintf(&b, "%s:%d:%d ", d.File, d.Line, d.Col)
	}
	if d.Struct != "" {
		if d.Field != "" {
			fmt.Fprintf(&b, "%s.%s ", d.Struct, d.Field)
		} else {
			fmt.Fprintf(&b, "%s ", d.Struct)
		}
	}
	if d.Directive != "" {
		fmt.Fprintf(&b, "%s ", d.Directive)
	}
	b.WriteString(d.Message)
	if d.Hint != "" {
		fmt.Fprintf(&b, " [hint: %s]", d.Hint)
	}
	return b.String()
}

// Result — the outcome of a generation run.
type Result struct {
	Written     []string // .inpupa.gen.go files written
	Unchanged   []string // .inpupa.gen.go files already up to date
	Deleted     []string // stale .inpupa.gen.go files removed
	Diagnostics []Diagnostic
}

// HasErrors reports whether any error diagnostics were produced.
func (r *Result) HasErrors() bool { return r.Count(SevError) > 0 }

// Count returns the number of diagnostics of the given severity.
func (r *Result) Count(sev Severity) int {
	n := 0
	for _, d := range r.Diagnostics {
		if d.Severity == sev {
			n++
		}
	}
	return n
}

// Generate runs generation for all packages under opts.InputDir.
// The returned error is fatal (bad input): the directory does not exist, go.mod is not found.
func Generate(opts Options) (*Result, error) {
	res := &Result{}
	abs, err := filepath.Abs(opts.InputDir)
	if err != nil {
		return res, fmt.Errorf("input directory: %w", err)
	}
	st, err := os.Stat(abs)
	if err != nil || !st.IsDir() {
		return res, fmt.Errorf("input directory %q is not a directory", opts.InputDir)
	}
	modPath, modRoot, err := moduleInfo(abs)
	if err != nil {
		return res, err
	}

	logf(opts.Verbose, "module %s (root %s)", modPath, modRoot)
	for _, rel := range discoverPackages(abs) {
		if excluded(rel, opts.Exclude) {
			continue
		}
		dir := filepath.Join(abs, rel)
		logf(opts.Verbose, "package %s", dir)
		generatePackage(res, opts, dir)
	}
	return res, nil
}

// moduleInfo finds the nearest go.mod at or above dir and returns the module path and root directory.
func moduleInfo(dir string) (modPath, modRoot string, err error) {
	d := dir
	for {
		gm := filepath.Join(d, "go.mod")
		if data, e := os.ReadFile(gm); e == nil {
			mod, ok := parseModulePath(data)
			if !ok {
				return "", "", fmt.Errorf("go.mod in %s: module directive not found", d)
			}
			return mod, d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", "", fmt.Errorf("no go.mod above directory %q — the generator must run inside a Go module", dir)
		}
		d = parent
	}
}

// parseModulePath extracts the module path from go.mod content.
func parseModulePath(data []byte) (string, bool) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		f := strings.Fields(line)
		if f[0] == "module" && len(f) >= 2 {
			return strings.Trim(f[1], `"'`), true
		}
	}
	return "", false
}

// discoverPackages returns sorted relative paths of directories containing Go source files
// (excluding _test.go and *.inpupa.gen.go). Service directories are skipped.
func discoverPackages(root string) []string {
	var dirs []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // a single unreadable file is not fatal for the whole tree
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != root && (name == "testdata" || name == "vendor" || isIgnoredDirName(name)) {
			return filepath.SkipDir
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		hasSource := false
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			n := e.Name()
			if !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") || isGenName(n) {
				continue
			}
			hasSource = true
			break
		}
		if hasSource {
			rel, _ := filepath.Rel(root, path)
			dirs = append(dirs, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(dirs)
	return dirs
}

// isIgnoredDirName — directories ignored by Go tooling.
func isIgnoredDirName(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

// isGenName reports whether the file name is a generated one.
func isGenName(name string) bool { return strings.HasSuffix(name, GenFileSuffix) }

// genPathFor returns the path of the generated file for a source file.
func genPathFor(srcFile string) string {
	return strings.TrimSuffix(srcFile, ".go") + GenFileSuffix
}

// excluded reports whether a package (rel path from the scan root) is excluded.
func excluded(rel string, excludes []string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, e := range excludes {
		e = filepath.ToSlash(filepath.Clean(strings.TrimPrefix(e, "./")))
		if e == "." || e == "" {
			continue
		}
		if rel == e || strings.HasPrefix(rel, e+"/") {
			return true
		}
	}
	return false
}

func logf(v bool, format string, args ...any) {
	if v {
		fmt.Fprintf(os.Stderr, "inpupa: "+format+"\n", args...)
	}
}
