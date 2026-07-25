// Command mwl is the command-line tool for the Metolia Workflow Language.
// See DESIGN.md in this directory for the decisions behind the surface:
// stderr carries text diagnostics for humans, stdout carries the requested
// JSON artifact or nothing, and exit codes separate the document's verdict
// (0 parsed, 1 rejected) from the tool's own failures (2).
package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/Element84/metolia-spec/parser"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

const rootUsage = `mwl - tooling for the Metolia Workflow Language

Usage:
  mwl <command> [arguments]

Commands:
  parse    check that a document parses; optionally emit its AST

Flags:
  --help     show this help
  --version  print the version

Use "mwl parse --help" for details on a command.
`

const parseUsage = `Usage:
  mwl parse flow <file|-> [--ast] [--strict]
  mwl parse provider <file|-> [--ast] [--strict]

Parse a document and report diagnostics. Diagnostics print to stderr as
"file:line:col: severity: message [code]", one per line, in document order;
a clean parse prints nothing. "-" reads the document from stdin.

Flags:
  --ast     emit a {"filename", "ast", "diagnostics"} JSON envelope on
            stdout; "ast" is null when the parse failed; "filename" is
            omitted for stdin
  --strict  exit 1 on any diagnostic, warnings included

Exit codes:
  0  the document parsed (warnings may have printed)
  1  the document was rejected, or produced any diagnostic under --strict
  2  the tool failed to run: unreadable input, unknown flag, bad usage

Examples:
  mwl parse flow flow.json
  git show HEAD:flow.json | mwl parse flow -
  mwl parse flow flow.json --ast | jq .ast
`

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, rootUsage)
		return 2
	}
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Fprint(stdout, rootUsage)
		return 0
	case "--version":
		fmt.Fprintf(stdout, "mwl %s\n", versionString())
		return 0
	case "parse":
		return runParse(args[1:], stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "mwl: unknown command %q\n\n", args[0])
		fmt.Fprint(stderr, rootUsage)
		return 2
	}
}

func runParse(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, parseUsage)
		return 2
	}
	switch args[0] {
	case "--help", "-h":
		fmt.Fprint(stdout, parseUsage)
		return 0
	case "flow", "provider":
		return runParseDoc(args[0], args[1:], stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "mwl parse: unknown document kind %q (want \"flow\" or \"provider\")\n\n", args[0])
		fmt.Fprint(stderr, parseUsage)
		return 2
	}
}

func runParseDoc(kind string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var ast, strict bool
	var input string
	var haveInput bool
	for _, a := range args {
		switch a {
		case "--ast":
			ast = true
		case "--strict":
			strict = true
		case "--help", "-h":
			fmt.Fprint(stdout, parseUsage)
			return 0
		default:
			if len(a) > 1 && a[0] == '-' {
				fmt.Fprintf(stderr, "mwl parse %s: unknown flag %q\n\n", kind, a)
				fmt.Fprint(stderr, parseUsage)
				return 2
			}
			if haveInput {
				fmt.Fprintf(stderr, "mwl parse %s: unexpected extra argument %q\n\n", kind, a)
				fmt.Fprint(stderr, parseUsage)
				return 2
			}
			input, haveInput = a, true
		}
	}
	if !haveInput {
		fmt.Fprint(stderr, parseUsage)
		return 2
	}

	var src []byte
	var err error
	var opts []parser.Option
	filename := ""
	if input == "-" {
		src, err = io.ReadAll(stdin)
	} else {
		src, err = os.ReadFile(input)
		filename = input
		opts = append(opts, parser.WithFilename(input))
	}
	if err != nil {
		fmt.Fprintf(stderr, "mwl parse %s: %v\n", kind, err)
		return 2
	}

	// The encoded AST for the envelope; nil renders as JSON null, the
	// failed-parse form.
	var encoded *obj
	var diags parser.Diagnostics
	switch kind {
	case "flow":
		var flow *parser.Flow
		flow, diags = parser.ParseFlow(src, opts...)
		if flow != nil {
			encoded = encFlow(flow)
		}
	case "provider":
		var prov *parser.Provider
		prov, diags = parser.ParseProvider(src, opts...)
		if prov != nil {
			encoded = encProvider(prov)
		}
	}

	for _, d := range diags {
		fmt.Fprintln(stderr, d.String())
	}
	if ast {
		if err := writeEnvelope(stdout, filename, encoded, diags); err != nil {
			fmt.Fprintf(stderr, "mwl parse %s: writing envelope: %v\n", kind, err)
			return 2
		}
	}

	if diags.HasErrors() {
		return 1
	}
	if strict && len(diags) > 0 {
		return 1
	}
	return 0
}

func versionString() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "(devel)"
}
