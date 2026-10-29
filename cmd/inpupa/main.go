// The inpupa generator emits FromRequest methods for structs annotated with @input:*.
//
// Usage from go:generate:
//
//	//go:generate inpupa --input .
package main

import (
	"os"

	"github.com/borodyadka/inpupa/internal/cli"
)

func main() { os.Exit(cli.Run(os.Args[1:])) }
