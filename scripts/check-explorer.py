#!/usr/bin/env python3
"""Verify the AST explorer page against the parser's AST declarations.

Extracts every exported Type.Field pair from the structs declared in the
parser's AST files — the two grammar files (parser/ast_flow.go,
parser/ast_provider.go), the shared carrier files (parser/ast.go,
parser/value.go), and Position alone from parser/diagnostic.go, whose other
types are the diagnostics API rather than AST vocabulary. Interfaces, methods,
constants, and unexported fields are out of scope; Pos and Comment fields are
covered by the trees' intro prose and ignored on both sides. From
content/tools/ast-explorer.md it extracts every data-field token
(space-separated Type.Field pairs) and every data-type value, with HTML
comments stripped so the authoring templates don't count.

Fails when a Go field has no covering data-field token, a token names a type
or field the parser doesn't declare, a data-type names an unknown type, or a
declared struct appears nowhere on the page (neither as a data-type header nor
inside any data-field token).

Usage:
    python3 scripts/check-explorer.py     # run from the site root

Exit status: 0 when clean, 1 on any discrepancy, 2 when a source file is
missing or a struct body has a line this parser does not recognize.
"""

import re
import sys
from pathlib import Path

# file -> restriction to these struct names, or None for every struct
AST_FILES: dict[Path, set[str] | None] = {
    Path("parser/ast_flow.go"): None,
    Path("parser/ast_provider.go"): None,
    Path("parser/ast.go"): None,
    Path("parser/value.go"): None,
    Path("parser/diagnostic.go"): {"Position"},
}
PAGE = Path("content/tools/ast-explorer.md")
IGNORED_FIELDS = {"Pos", "Comment"}

# Optional [T any]-style parameter list after the name (OrExpr).
STRUCT_OPEN = re.compile(r"^type (\w+)(?:\[[^\]]+\])? struct \{$")
# One or more comma-separated field names, then a type. Trailing // comments
# are stripped before matching.
FIELD = re.compile(r"^\t(\w+(?:, \w+)*)\s+\S")


def parse_structs(path: Path) -> dict[str, list[str]]:
    """Map struct name -> exported field names, in declaration order."""
    structs: dict[str, list[str]] = {}
    current = None
    for lineno, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        if current is None:
            m = STRUCT_OPEN.match(line)
            if m:
                current = m.group(1)
                structs[current] = []
            continue
        if line == "}":
            current = None
            continue
        stripped = line.strip()
        if not stripped or stripped.startswith("//"):
            continue
        code = line.split("//", 1)[0].rstrip()
        m = FIELD.match(code)
        if not m:
            print(
                f"error: {path}:{lineno}: unrecognized line in struct "
                f"{current}: {stripped!r}",
                file=sys.stderr,
            )
            sys.exit(2)
        for name in (n.strip() for n in m.group(1).split(",")):
            if name[0].isupper() and name not in IGNORED_FIELDS:
                structs[current].append(name)
    return structs


def main() -> int:
    for path in [*AST_FILES, PAGE]:
        if not path.is_file():
            print(f"error: {path} not found; run from the site root", file=sys.stderr)
            return 2

    structs: dict[str, list[str]] = {}
    for path, only in AST_FILES.items():
        parsed = parse_structs(path)
        if only is not None:
            parsed = {t: f for t, f in parsed.items() if t in only}
        structs.update(parsed)
    declared = {(t, f) for t, fields in structs.items() for f in fields}

    text = re.sub(r"<!--.*?-->", "", PAGE.read_text(encoding="utf-8"), flags=re.S)
    field_tokens = [
        t for v in re.findall(r'data-field="([^"]*)"', text) for t in v.split()
    ]
    type_values = re.findall(r'data-type="([^"]*)"', text)

    covered = set()
    bad_tokens = []
    for token in field_tokens:
        parts = token.split(".")
        if len(parts) != 2:
            bad_tokens.append(f"{token} (malformed: want Type.Field)")
            continue
        t, f = parts
        if f in IGNORED_FIELDS:
            bad_tokens.append(f"{token} ({f} is covered by intro prose, not rows)")
            continue
        if t not in structs:
            bad_tokens.append(f"{token} (no struct {t})")
        elif f not in structs[t]:
            bad_tokens.append(f"{token} (struct {t} has no field {f})")
        else:
            covered.add((t, f))

    missing = sorted(declared - covered)
    unknown_types = sorted(t for t in type_values if t not in structs)
    on_page = set(type_values) | {t for t, _ in covered}
    absent_structs = sorted(t for t in structs if t not in on_page)

    problems = len(missing) + len(bad_tokens) + len(unknown_types) + len(absent_structs)

    print(f"explorer: {PAGE}")
    print(f"  declared fields:   {len(declared)} across {len(structs)} structs")
    print(f"  data-field tokens: {len(field_tokens)}")
    print(f"  data-type rows:    {len(type_values)}")
    print(f"  discrepancies:     {problems}")

    if missing:
        print("\nFIELDS WITH NO COVERING data-field TOKEN:")
        for t, f in missing:
            print(f"  {t}.{f}")
    if bad_tokens:
        print("\ndata-field TOKENS THE PARSER DOES NOT DECLARE:")
        for token in bad_tokens:
            print(f"  {token}")
    if unknown_types:
        print("\ndata-type VALUES NAMING NO DECLARED STRUCT:")
        for t in unknown_types:
            print(f"  {t}")
    if absent_structs:
        print("\nDECLARED STRUCTS ABSENT FROM THE PAGE:")
        for t in absent_structs:
            print(f"  {t}")

    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
