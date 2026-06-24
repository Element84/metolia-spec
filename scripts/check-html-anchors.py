#!/usr/bin/env python3
"""Verify fragment anchors in the rendered HTML site.

Walks the rendered site under public/ and checks that every internal link
carrying a #fragment points at an element id that exists on the target page.
This catches breakage Hugo's --panicOnWarning cannot: Hugo does not validate
fragments, and its heading slugger differs from the GitHub-style slugs other
tools assume (review finding F8 shipped exactly such a mismatch).

Usage:
    hugo                                        # render to public/ first
    python3 scripts/check-html-anchors.py [section ...]

With no arguments every page's outgoing links are checked. Passing section
names (e.g. `reference`) restricts which pages' OUTGOING links are
checked; link targets may resolve anywhere in the site.

Exit status: 0 when clean, 1 when any broken fragment or missing target page
is found, 2 on usage errors.
"""

import re
import sys
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import unquote, urldefrag, urljoin

PUBLIC = Path("public")
SCHEME = re.compile(r"^[a-z][a-z0-9+.-]*:")


class PageScan(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.ids = set()
        self.hrefs = []

    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        if a.get("id"):
            self.ids.add(a["id"])
        if tag == "a":
            if a.get("name"):
                self.ids.add(a["name"])
            if a.get("href"):
                self.hrefs.append(a["href"])


def url_for(html_file: Path) -> str:
    rel = html_file.relative_to(PUBLIC).as_posix()
    if rel.endswith("/index.html"):
        return "/" + rel[: -len("index.html")]
    if rel == "index.html":
        return "/"
    return "/" + rel


def main() -> int:
    if not PUBLIC.is_dir():
        print("error: public/ not found; run `hugo` first", file=sys.stderr)
        return 2
    sections = sys.argv[1:]

    pages = {}  # url -> PageScan
    for f in sorted(PUBLIC.rglob("*.html")):
        scan = PageScan()
        scan.feed(f.read_text(encoding="utf-8", errors="replace"))
        pages[url_for(f)] = scan

    def in_scope(url: str) -> bool:
        if not sections:
            return True
        return any(url.startswith(f"/{s.strip('/')}/") for s in sections)

    broken = []  # (page, href, reason)
    n_checked = 0
    for url, scan in pages.items():
        if not in_scope(url):
            continue
        for href in scan.hrefs:
            if "#" not in href or href.startswith("//") or SCHEME.match(href):
                continue
            target, frag = urldefrag(urljoin(url, href))
            if not frag:
                continue
            n_checked += 1
            if not target.endswith("/") and target + "/" in pages:
                target += "/"
            tgt = pages.get(target)
            if tgt is None:
                broken.append((url, href, f"target page not found: {target}"))
            elif unquote(frag) not in tgt.ids:
                broken.append((url, href, f"no id '{frag}' on {target}"))

    scope = ", ".join(sections) if sections else "whole site"
    print(f"scope: {scope}")
    print(f"  pages scanned:         {len(pages)}")
    print(f"  fragment links checked: {n_checked}")
    print(f"  broken:                 {len(broken)}")
    if broken:
        print("\nBROKEN FRAGMENT LINKS:")
        for page, href, reason in broken:
            print(f"  {page}\n    {href}\n    -> {reason}")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
