// Package cli implements the command line interface of the inpupa generator.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/borodyadka/inpupa/internal/gen"
)

// Run performs generation and returns the exit code:
// 0 — success (warnings allowed); 1 — validation errors (or warnings with --werror);
// 2 — CLI or environment errors.
func Run(args []string) int {
	fs := pflag.NewFlagSet("inpupa", pflag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	input := fs.String("input", ".", "directory to scan for annotated packages (recursive)")
	exclude := fs.StringSlice("exclude", nil, "directories relative to --input to skip entirely (comma-separated, may be repeated)")
	verbose := fs.Bool("verbose", false, "verbose logging on stderr")
	werror := fs.Bool("werror", false, "treat warnings as errors (exit code 1)")
	if err := fs.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return 0
		}
		return 2
	}

	res, err := gen.Generate(gen.Options{InputDir: *input, Exclude: *exclude, Verbose: *verbose})
	if err != nil {
		fmt.Fprintf(os.Stderr, "inpupa: %v\n", err)
		return 2
	}
	for _, d := range res.Diagnostics {
		fmt.Fprintln(os.Stderr, d.Render())
	}
	logf(*verbose, "summary: written=%d unchanged=%d deleted=%d errors=%d warnings=%d",
		len(res.Written), len(res.Unchanged), len(res.Deleted), res.Count(gen.SevError), res.Count(gen.SevWarning))

	if res.HasErrors() || (*werror && res.Count(gen.SevWarning) > 0) {
		return 1
	}
	return 0
}

func logf(v bool, format string, args ...any) {
	if v {
		fmt.Fprintf(os.Stderr, "inpupa: "+format+"\n", args...)
	}
}
