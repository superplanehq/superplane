#!/usr/bin/env bash
#
# Lint proto message field numbers for gaps.
#
# House style: within each message, field numbers stay contiguous (1..N with no
# holes). The protos here are used for JSON conversion, not wire compatibility,
# so a removed field should be renumbered rather than left as a gap. This is a
# light heuristic, not a full proto parser: enum values and options are ignored
# because they lack a type token, and comments/strings are stripped before
# scanning.

set -euo pipefail
shopt -s nullglob

# Run from the repo root so the protos/*.proto glob resolves the same way no
# matter where make invokes us from.
cd "$(dirname "$0")/.."

files=(protos/*.proto)
if [ ${#files[@]} -eq 0 ]; then
	echo "No proto files found under protos/." >&2
	exit 1
fi

awk '
	# Reset the block stack at the start of each file.
	FNR == 1 {
		sp = 0
		delete kind; delete nm; delete lo; delete hi; delete cnt; delete nums
	}

	{
		line = $0
		gsub(/\/\/[^\n]*/, "", line)   # line comments
		gsub(/"[^"]*"/, "", line)       # string literals (may hold braces)

		stripped = line
		sub(/^[ \t]+/, "", stripped)

		# A message field line: an optional label, a type (or map<...>), a name,
		# then "= N". Enum values ("NAME = N") and options have no type token, so
		# they never match and are correctly ignored.
		if (stripped ~ /^(repeated|optional|required)?[ \t]*(map[ \t]*<[^>]*>|[A-Za-z0-9_.]+)[ \t]+[A-Za-z0-9_]+[ \t]*=[ \t]*[0-9]+/) {
			seg = stripped
			match(seg, /=[ \t]*[0-9]+/)
			num = substr(seg, RSTART, RLENGTH)
			gsub(/[^0-9]/, "", num)
			num = num + 0
			# oneof fields belong to the enclosing message.
			for (d = sp; d >= 1; d--) {
				if (kind[d] == "message") {
					nums[d, num] = 1
					if (cnt[d] == 0 || num < lo[d]) lo[d] = num
					if (cnt[d] == 0 || num > hi[d]) hi[d] = num
					cnt[d]++
					break
				}
			}
		}

		n = length(line)
		for (i = 1; i <= n; i++) {
			c = substr(line, i, 1)
			if (c == "{") {
				sp++
				if (match(stripped, /^(message|enum|oneof)[ \t]+[A-Za-z0-9_]+/)) {
					opener = substr(stripped, RSTART, RLENGTH)
					split(opener, parts, /[ \t]+/)
					kind[sp] = parts[1]
					nm[sp] = parts[2]
				} else {
					kind[sp] = "block"
					nm[sp] = ""
				}
				lo[sp] = ""; hi[sp] = ""; cnt[sp] = 0
			} else if (c == "}" && sp > 0) {
				d = sp
				sp--
				if (kind[d] == "message" && cnt[d] > 0) {
					missing = ""
					for (v = lo[d]; v <= hi[d]; v++) {
						if (!((d SUBSEP v) in nums)) {
							missing = missing (missing == "" ? "" : ", ") v
						}
					}
					if (missing != "") {
						qualified = ""
						for (e = 1; e <= d; e++) {
							if (kind[e] == "message") {
								qualified = qualified (qualified == "" ? "" : ".") nm[e]
							}
						}
						printf "%s: %s: field numbers have gaps: missing %s\n", FILENAME, qualified, missing > "/dev/stderr"
						bad = 1
					}
				}
				# Drop this frame'\''s field numbers so a sibling reusing the depth
				# starts clean.
				for (v = lo[d]; v <= hi[d]; v++) delete nums[d, v]
			}
		}
	}

	END {
		if (bad) {
			print "" > "/dev/stderr"
			print "Proto message field numbers must be contiguous with no gaps." > "/dev/stderr"
			print "Renumber the remaining fields so the numbers run 1..N, then run" > "/dev/stderr"
			print "\"make pb.gen\". Do not use \"reserved\" to paper over a hole." > "/dev/stderr"
			exit 1
		}
		print "Proto message field numbers are contiguous."
	}
' "${files[@]}"
