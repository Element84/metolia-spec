package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files")

// TestGolden runs the CLI in-process over the testdata fixtures and compares
// both streams byte-for-byte against committed golden files. The goldens are
// the executable documentation of the --ast envelope (see DESIGN.md); after
// a deliberate encoding change, regenerate with: go test ./... -update
func TestGolden(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		stdin    string // fixture path to feed as stdin; empty for none
		wantExit int
	}{
		{name: "raise", args: []string{"parse", "flow", "testdata/flow/raise.json"}, wantExit: 0},
		{name: "raise-ast", args: []string{"parse", "flow", "testdata/flow/raise.json", "--ast"}, wantExit: 0},
		{name: "match-ast", args: []string{"parse", "flow", "testdata/flow/match.json", "--ast"}, wantExit: 0},
		{name: "gather-scatter-ast", args: []string{"parse", "flow", "testdata/flow/gather-scatter.json", "--ast"}, wantExit: 0},
		{name: "gather-iterate-ast", args: []string{"parse", "flow", "testdata/flow/gather-iterate.json", "--ast"}, wantExit: 0},
		{name: "subflow-ast", args: []string{"parse", "flow", "testdata/flow/subflow.json", "--ast"}, wantExit: 0},
		{name: "middleware-ast", args: []string{"parse", "flow", "testdata/flow/middleware.json", "--ast"}, wantExit: 0},
		{name: "invalid", args: []string{"parse", "flow", "testdata/flow/invalid-unresolved.json"}, wantExit: 1},
		{name: "invalid-ast", args: []string{"parse", "flow", "testdata/flow/invalid-unresolved.json", "--ast"}, wantExit: 1},
		{name: "warning", args: []string{"parse", "flow", "testdata/flow/warning-shadowed.json"}, wantExit: 0},
		{name: "warning-strict", args: []string{"parse", "flow", "testdata/flow/warning-shadowed.json", "--strict"}, wantExit: 1},
		{name: "warning-ast", args: []string{"parse", "flow", "testdata/flow/warning-shadowed.json", "--ast"}, wantExit: 0},
		{name: "stdin-ast", args: []string{"parse", "flow", "-", "--ast"}, stdin: "testdata/flow/raise.json", wantExit: 0},
		{name: "provider-call-ast", args: []string{"parse", "provider", "testdata/provider/call.json", "--ast"}, wantExit: 0},
		{name: "provider-middleware-ast", args: []string{"parse", "provider", "testdata/provider/middleware.json", "--ast"}, wantExit: 0},
		{name: "provider-invalid", args: []string{"parse", "provider", "testdata/provider/invalid-catalog.json"}, wantExit: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdin bytes.Buffer
			if tc.stdin != "" {
				src, err := os.ReadFile(tc.stdin)
				if err != nil {
					t.Fatal(err)
				}
				stdin.Write(src)
			}
			var stdout, stderr bytes.Buffer
			exit := run(tc.args, &stdin, &stdout, &stderr)
			if exit != tc.wantExit {
				t.Errorf("exit = %d, want %d\nstderr:\n%s", exit, tc.wantExit, stderr.String())
			}
			checkGolden(t, tc.name+".stdout", stdout.Bytes())
			checkGolden(t, tc.name+".stderr", stderr.Bytes())
		})
	}
}

// checkGolden compares got against testdata/golden/<name>; with -update it
// rewrites the file instead (and removes it when got is empty, so empty
// streams need no file at all).
func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if *update {
		if len(got) == 0 {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			return
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		want = nil
	} else if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs from golden file %s\ngot:\n%s\nwant:\n%s", name, path, got, want)
	}
}

func TestToolErrors(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantExit   int
		wantStderr string // substring
	}{
		{name: "no-args", args: nil, wantExit: 2, wantStderr: "Usage"},
		{name: "unknown-command", args: []string{"frobnicate"}, wantExit: 2, wantStderr: "unknown command"},
		{name: "parse-no-args", args: []string{"parse"}, wantExit: 2, wantStderr: "Usage"},
		{name: "unknown-kind", args: []string{"parse", "yaml", "x.json"}, wantExit: 2, wantStderr: "unknown document kind"},
		{name: "provider-no-input", args: []string{"parse", "provider"}, wantExit: 2, wantStderr: "Usage"},
		{name: "provider-missing-file", args: []string{"parse", "provider", "testdata/no-such-file.json"}, wantExit: 2, wantStderr: "no-such-file.json"},
		{name: "missing-file", args: []string{"parse", "flow", "testdata/no-such-file.json"}, wantExit: 2, wantStderr: "no-such-file.json"},
		{name: "unknown-flag", args: []string{"parse", "flow", "x.json", "--verbose"}, wantExit: 2, wantStderr: "unknown flag"},
		{name: "extra-arg", args: []string{"parse", "flow", "a.json", "b.json"}, wantExit: 2, wantStderr: "extra argument"},
		{name: "flow-no-input", args: []string{"parse", "flow"}, wantExit: 2, wantStderr: "Usage"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exit := run(tc.args, strings.NewReader(""), &stdout, &stderr)
			if exit != tc.wantExit {
				t.Errorf("exit = %d, want %d", exit, tc.wantExit)
			}
			if !strings.Contains(stderr.String(), tc.wantStderr) {
				t.Errorf("stderr %q does not contain %q", stderr.String(), tc.wantStderr)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout not empty on tool error: %q", stdout.String())
			}
		})
	}
}

func TestHelpAndVersion(t *testing.T) {
	for _, args := range [][]string{
		{"--help"},
		{"help"},
		{"parse", "--help"},
		{"parse", "flow", "--help"},
		{"parse", "provider", "--help"},
		{"--version"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exit := run(args, strings.NewReader(""), &stdout, &stderr)
			if exit != 0 {
				t.Errorf("exit = %d, want 0", exit)
			}
			if stdout.Len() == 0 {
				t.Error("expected help/version text on stdout")
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr not empty: %q", stderr.String())
			}
		})
	}
}
