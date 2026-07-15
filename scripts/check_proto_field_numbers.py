#!/usr/bin/env python3
"""Lint proto message field numbers for gaps.

House style: within each message, field numbers stay contiguous (1..N with no
holes). The protos here are used for JSON conversion, not wire compatibility, so
a removed field should be renumbered rather than left as a gap. This is a light
heuristic, not a full proto parser: enum values and options are ignored because
they lack a type token, and comments/strings are stripped before scanning.
"""
import glob
import re
import sys

# A message field line: an optional label, a type (or map<...>), a name, then
# "= N". Enum values ("NAME = N") and options have no type token, so they never
# match and are correctly ignored.
FIELD = re.compile(
    r"^(?:repeated|optional|required)?\s*"
    r"(?:map\s*<[^>]*>|[\w.]+)\s+\w+\s*=\s*(\d+)\b"
)
OPENER = re.compile(r"^(message|enum|oneof)\s+(\w+)")


def check(path):
    src = open(path).read()
    src = re.sub(r"//[^\n]*", "", src)          # line comments
    src = re.sub(r'"[^"\n]*"', "", src)          # string literals (may hold {})

    issues = []
    stack = []  # frames: {"kind", "name", "nums"}
    for line in src.splitlines():
        stripped = line.strip()

        field = FIELD.match(stripped)
        if field:
            for frame in reversed(stack):        # oneof fields belong to the message
                if frame["kind"] == "message":
                    frame["nums"].add(int(field.group(1)))
                    break

        for char in line:
            if char == "{":
                opener = OPENER.match(stripped)
                kind = opener.group(1) if opener else "block"
                name = opener.group(2) if opener else ""
                stack.append({"kind": kind, "name": name, "nums": set()})
            elif char == "}" and stack:
                frame = stack.pop()
                if frame["kind"] == "message" and frame["nums"]:
                    lo, hi = min(frame["nums"]), max(frame["nums"])
                    missing = [n for n in range(lo, hi + 1) if n not in frame["nums"]]
                    if missing:
                        qualified = ".".join(
                            f["name"] for f in stack + [frame] if f["kind"] == "message"
                        )
                        issues.append((path, qualified, missing))
    return issues


def main():
    issues = []
    for path in sorted(glob.glob("protos/*.proto")):
        issues.extend(check(path))

    if not issues:
        print("Proto message field numbers are contiguous.")
        return 0

    for path, message, missing in issues:
        nums = ", ".join(str(n) for n in missing)
        print(f"{path}: {message}: field numbers have gaps: missing {nums}", file=sys.stderr)
    print(
        "\nProto message field numbers must be contiguous with no gaps.\n"
        "Renumber the remaining fields so the numbers run 1..N, then run "
        '"make pb.gen". Do not use "reserved" to paper over a hole.',
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    sys.exit(main())
