#!/usr/bin/env bash
#
# Static guard for JS/TS tool configs that CI and editors execute.
# Read file bytes only. Do not import, eval, or run ESLint/Node on the files.
#
# Catches the PR 7318 PolinRider pattern: createRequire / node:module in an
# ESLint config, a packed payload after a long run of spaces, or a huge last line.

set -euo pipefail

cd "$(dirname "$0")/.."
root="${1:-.}"

# 500 chars is far above any legitimate config line in this repo.
# The PR 7318 dropper hid about 8000 characters on the last line.
max_line_length=500
# Diff viewers clip a line that starts with thousands of spaces.
max_space_run=200

# Patterns that a clean SuperPlane tool config does not need.
# node:path and node:url stay allowed (Storybook uses them).
forbidden_regex='createRequire|node:module|child_process|eval\(|new Function|Function\(|global\.o='

candidates=(
	web_src/eslint.config.js
	web_src/eslint.config.mjs
	web_src/eslint.config.cjs
	web_src/vite.config.ts
	web_src/vite.config.js
	web_src/vitest.config.ts
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

files=()
for rel in "${candidates[@]}"; do
	if [ -f "$root/$rel" ]; then
		files+=("$root/$rel")
	fi
done

if [ "${#files[@]}" -eq 0 ]; then
	echo "No tool config files found under $root" >&2
	exit 1
fi

failed=0

for file in "${files[@]}"; do
	if grep -Eq "$forbidden_regex" "$file"; then
		echo "FORBIDDEN token in $file (createRequire, node:module, child_process, eval, Function, or global.o)" >&2
		failed=1
	fi

	if LC_ALL=C grep -Eq "[[:space:]]{$max_space_run,}" "$file"; then
		echo "Long whitespace run (>= $max_space_run) in $file" >&2
		failed=1
	fi

	line_no=0
	while IFS= read -r line || [ -n "$line" ]; do
		line_no=$((line_no + 1))
		len=${#line}
		if [ "$len" -gt "$max_line_length" ]; then
			echo "Line $line_no in $file is $len chars (max $max_line_length)" >&2
			failed=1
		fi
	done <"$file"
done

if [ "$failed" -ne 0 ]; then
	echo "Tool config guard failed. Do not run ESLint or Node on these files." >&2
	exit 1
fi

echo "Tool config guard passed (${#files[@]} files)."
