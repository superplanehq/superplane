#!/usr/bin/env bash
#
# Static guard for JS/TS tool configs that CI and editors execute.
# Read file bytes only. Do not import, eval, or run ESLint/Node on the files.

set -euo pipefail

cd "$(dirname "$0")/.."
root="${1:-.}"

max_line_length=500
max_space_run=200

# Literal dropper tokens plus the compact aliases Greptile flagged.
# node:path and node:url stay allowed (Storybook uses them).
forbidden_regex='createRequire|node:module|child_process|eval\(|new Function|Function\(|global\.o=|globalThis\[|require\(|Function\.constructor|['\''"]eval['\''"]|['\''"]Function['\''"]|['\''"]child['\''"][[:space:]]*\+'

candidates=(
	web_src/eslint.config.js
	web_src/eslint.config.mjs
	web_src/eslint.config.cjs
	web_src/vite.config.ts
	web_src/vite.config.js
	web_src/vitest.config.ts
	web_src/openapi-ts.config.ts
	web_src/.storybook/main.ts
	web_src/.storybook/manager.ts
	web_src/.eslintrc.js
	web_src/.eslintrc.cjs
	web_src/prettier.config.js
	web_src/prettier.config.cjs
	web_src/prettier.config.mjs
	web_src/postcss.config.js
	web_src/postcss.config.cjs
)

echo "==> Tool config guard"
echo "    Static byte scan. Does not load ESLint or Node configs."

files=()
for rel in "${candidates[@]}"; do
	if [ -f "$root/$rel" ]; then
		files+=("$root/$rel")
	fi
done

if [ "${#files[@]}" -eq 0 ]; then
	echo "FAIL  no tool config files under $root" >&2
	exit 1
fi

failed=0

for file in "${files[@]}"; do
	rel="${file#"$root"/}"
	file_fail=0
	reasons=()

	if grep -Eq "$forbidden_regex" "$file"; then
		file_fail=1
		reasons+=("forbidden token (createRequire, eval alias, require, or dropper marker)")
	fi

	if LC_ALL=C grep -Eq "[[:space:]]{$max_space_run,}" "$file"; then
		file_fail=1
		reasons+=("whitespace run >= $max_space_run")
	fi

	line_no=0
	longest=0
	while IFS= read -r line || [ -n "$line" ]; do
		line_no=$((line_no + 1))
		len=${#line}
		if [ "$len" -gt "$longest" ]; then
			longest=$len
		fi
		if [ "$len" -gt "$max_line_length" ]; then
			file_fail=1
			reasons+=("line $line_no is $len chars (max $max_line_length)")
		fi
	done <"$file"

	if [ "$file_fail" -ne 0 ]; then
		failed=1
		echo "FAIL  $rel"
		for reason in "${reasons[@]}"; do
			echo "      $reason"
		done
	else
		echo "OK    $rel  lines=$line_no  longest=$longest"
	fi
done

if [ "$failed" -ne 0 ]; then
	echo "==> Tool config guard: FAIL"
	echo "    Do not run ESLint or Node on the failed files."
	exit 1
fi

echo "==> Tool config guard: PASS (${#files[@]} files)"
