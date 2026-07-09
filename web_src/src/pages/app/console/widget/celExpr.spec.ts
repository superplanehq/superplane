import { describe, it, expect } from "vitest";

import {
  buildEnv,
  compileExpr,
  compileMaybeExpr,
  compileTemplate,
  evalExpr,
  evalExprDetailed,
  evalRowField,
  evalTemplate,
  evalTemplateDetailed,
} from "./celExpr";
import { getValueAtPath } from "./fieldPath";

describe("celExpr", () => {
  it("evaluates a simple CEL expression against row fields", () => {
    const maybe = compileMaybeExpr('{{ status == "running" }}');
    expect(maybe.kind).toBe("expr");
    if (maybe.kind !== "expr") return;
    const env = buildEnv();
    const row = { status: "running" };
    expect(evalExpr(maybe.expr, row, env)).toBe(true);
  });

  it("resolves literal dot paths", () => {
    const maybe = compileMaybeExpr("pr_number");
    const env = buildEnv();
    expect(evalRowField(maybe, { pr_number: "42" }, env, getValueAtPath)).toBe("42");
  });

  it("interpolates templates", () => {
    const template = compileTemplate("Destroy PR #{{ pr_number }}?");
    const env = buildEnv();
    expect(evalTemplate(template, { pr_number: "69" }, env, String)).toBe("Destroy PR #69?");
  });

  describe("numeric-string coercion", () => {
    it("evaluates `value / 2` when value is a numeric string", () => {
      const compiled = compileExpr("value / 2");
      const env = buildEnv();
      expect(evalExpr(compiled, { value: "10" }, env)).toBe(5);
    });

    it("evaluates mixed arithmetic with stringified numbers", () => {
      const compiled = compileExpr("value * factor + offset");
      const env = buildEnv();
      expect(evalExpr(compiled, { value: "4", factor: "3", offset: "2" }, env)).toBe(14);
    });

    it("interpolates `{{ value / 2 }}` against stringified row values", () => {
      const template = compileTemplate("Half = {{ value / 2 }}");
      const env = buildEnv();
      expect(evalTemplate(template, { value: "10" }, env, String)).toBe("Half = 5");
    });

    it("returns undefined when the operand cannot be coerced to a number", () => {
      const compiled = compileExpr("value / 2");
      const env = buildEnv();
      expect(evalExpr(compiled, { value: "not a number" }, env)).toBeUndefined();
    });

    it("preserves string equality (does not coerce all strings unconditionally)", () => {
      const compiled = compileExpr('name == "42"');
      const env = buildEnv();
      expect(evalExpr(compiled, { name: "42" }, env)).toBe(true);
      expect(evalExpr(compiled, { name: "43" }, env)).toBe(false);
    });
  });

  describe("formatDate builtin", () => {
    it("formats an ISO timestamp using MM/dd in local time", () => {
      const local = new Date(2026, 2, 15, 14, 30); // March 15, 2026 (local TZ)
      const compiled = compileExpr('formatDate(createdAt, "MM/dd")');
      const env = buildEnv();
      expect(evalExpr(compiled, { createdAt: local.toISOString() }, env)).toBe("03/15");
    });

    it("supports yyyy-MM-dd HH:mm patterns", () => {
      const local = new Date(2026, 0, 5, 9, 7); // Jan 5, 2026 09:07
      const compiled = compileExpr('formatDate(ts, "yyyy-MM-dd HH:mm")');
      expect(evalExpr(compiled, { ts: local.toISOString() }, buildEnv())).toBe("2026-01-05 09:07");
    });

    it("accepts a Date instance and renders single-digit tokens unpadded", () => {
      const local = new Date(2026, 4, 3, 7, 4, 9); // May 3, 2026 07:04:09
      const compiled = compileExpr('formatDate(value, "M/d H:m:s")');
      expect(evalExpr(compiled, { value: local }, buildEnv())).toBe("5/3 7:4:9");
    });

    it("treats large numbers as epoch milliseconds", () => {
      const local = new Date(2026, 5, 1, 12, 0); // Jun 1, 2026 12:00
      const compiled = compileExpr('formatDate(ms, "yyyy")');
      expect(evalExpr(compiled, { ms: local.getTime() }, buildEnv())).toBe("2026");
    });

    it("treats small numbers as epoch seconds", () => {
      const local = new Date(2026, 6, 4, 12, 0); // Jul 4, 2026 12:00
      const seconds = Math.trunc(local.getTime() / 1000);
      const compiled = compileExpr('formatDate(sec, "MM/dd")');
      expect(evalExpr(compiled, { sec: seconds }, buildEnv())).toBe("07/04");
    });

    it("returns empty string for unparseable values and empty patterns", () => {
      const env = buildEnv();
      expect(evalExpr(compileExpr('formatDate(bad, "MM/dd")'), { bad: "not a date" }, env)).toBe("");
      expect(evalExpr(compileExpr('formatDate(value, "")'), { value: "2026-03-15T00:00:00Z" }, env)).toBe("");
      expect(evalExpr(compileExpr('formatDate(value, "MM/dd")'), { value: null }, env)).toBe("");
    });

    it("preserves non-token characters in the pattern", () => {
      const local = new Date(2026, 2, 15);
      const compiled = compileExpr('formatDate(value, "[yyyy]/[MM]")');
      expect(evalExpr(compiled, { value: local.toISOString() }, buildEnv())).toBe("[2026]/[03]");
    });
  });

  describe("epochMs builtin", () => {
    it("converts an ISO-8601 string to ms-since-epoch", () => {
      const compiled = compileExpr("epochMs(value)");
      const env = buildEnv();
      const result = evalExpr(compiled, { value: "2026-01-01T00:00:00Z" }, env) as number;
      expect(result).toBe(Date.UTC(2026, 0, 1, 0, 0, 0));
    });

    it("supports timestamp arithmetic across two ISO strings", () => {
      // Authors hit this when they write `{{ epochMs(finishedAt) - epochMs(createdAt) }}`
      // on a runs row to compute elapsed time without the durationMs convenience.
      const compiled = compileExpr("epochMs(finishedAt) - epochMs(createdAt)");
      const env = buildEnv();
      const result = evalExpr(compiled, { createdAt: "2026-01-01T12:00:00Z", finishedAt: "2026-01-01T12:05:00Z" }, env);
      expect(result).toBe(5 * 60 * 1000);
    });

    it("composes with `duration()` for human-friendly output", () => {
      const template = compileTemplate("Took {{ duration((epochMs(finishedAt) - epochMs(createdAt)) / 1000) }}");
      const env = buildEnv();
      const result = evalTemplate(
        template,
        { createdAt: "2026-01-01T12:00:00Z", finishedAt: "2026-01-01T12:05:00Z" },
        env,
        String,
      );
      expect(result).toBe("Took 5m");
    });

    it("returns 0 for unparseable inputs so arithmetic stays defined", () => {
      const compiled = compileExpr("epochMs(value)");
      const env = buildEnv();
      expect(evalExpr(compiled, { value: "not a date" }, env)).toBe(0);
      expect(evalExpr(compiled, { value: null }, env)).toBe(0);
    });

    it("accepts epoch numbers (seconds and ms) and Date instances", () => {
      const compiled = compileExpr("epochMs(value)");
      const env = buildEnv();
      const ms = Date.UTC(2026, 5, 1, 12, 0);
      expect(evalExpr(compiled, { value: ms }, env)).toBe(ms);
      expect(evalExpr(compiled, { value: ms / 1000 }, env)).toBe(ms);
      expect(evalExpr(compiled, { value: new Date(ms) }, env)).toBe(ms);
    });
  });

  describe("parseJson builtin", () => {
    // cel-js's grammar does not allow postfix `.foo` / `[i]` / `.method(...)`
    // after a function call result. So `parseJson(blob).items` is a parse
    // error. The builtin is still useful when composed with other functions
    // (`size(...)`, `string(...)`, equality), or when the whole expression
    // is `parseJson(value)` and the renderer consumes the structured result.
    it("parses a JSON array string and returns it wholesale", () => {
      const compiled = compileExpr("parseJson(tags)");
      expect(evalExpr(compiled, { tags: '["a","b"]' }, buildEnv())).toEqual(["a", "b"]);
    });

    it("parses a JSON object string and returns it wholesale", () => {
      const compiled = compileExpr("parseJson(blob)");
      const result = evalExpr(compiled, { blob: '{"items":[{"id":1}]}' }, buildEnv());
      expect(result).toEqual({ items: [{ id: 1 }] });
    });

    it("composes with size() to count parsed elements", () => {
      const compiled = compileExpr("size(parseJson(tags))");
      expect(evalExpr(compiled, { tags: '["a","b","c"]' }, buildEnv())).toBe(3);
    });

    it("passes already-parsed values through unchanged", () => {
      const compiled = compileExpr("size(parseJson(tags))");
      expect(evalExpr(compiled, { tags: ["a", "b", "c"] }, buildEnv())).toBe(3);
    });

    it("returns null for malformed JSON so equality checks stay defined", () => {
      const compiled = compileExpr("parseJson(bad) == null");
      expect(evalExpr(compiled, { bad: "not json" }, buildEnv())).toBe(true);
    });

    it("returns null for null inputs without throwing", () => {
      const compiled = compileExpr("parseJson(value)");
      expect(evalExpr(compiled, { value: null }, buildEnv())).toBeNull();
    });

    it("works inside templated interpolation when the whole expression is parseJson", () => {
      const template = compileTemplate("Tags: {{ parseJson(blob) }}");
      const result = evalTemplate(template, { blob: '["a","b"]' }, buildEnv(), String);
      // Template stringify uses String() on the parsed value; nested objects
      // serialize via JS's default Array#toString. Authors who need pretty
      // formatting should compose `string()` or shape the data upstream.
      expect(result).toBe("Tags: a,b");
    });
  });

  describe("string trimming builtins", () => {
    // These exist because cel-js doesn't allow postfix `[i]` / `.method()`
    // after a function-call result. The user-facing requirement is "first
    // line" / "first X chars" against runs data (raw multi-line outputs);
    // each helper returns the scalar directly so authors don't need
    // chaining the language can't parse.

    describe("firstLine", () => {
      it("returns the text before the first newline", () => {
        const compiled = compileExpr("firstLine(message)");
        expect(evalExpr(compiled, { message: "hello\nworld\nbye" }, buildEnv())).toBe("hello");
      });

      it("treats CRLF and bare CR the same as LF", () => {
        const compiled = compileExpr("firstLine(message)");
        expect(evalExpr(compiled, { message: "alpha\r\nbeta" }, buildEnv())).toBe("alpha");
        expect(evalExpr(compiled, { message: "alpha\rbeta" }, buildEnv())).toBe("alpha");
      });

      it("returns the input unchanged when there is no newline", () => {
        const compiled = compileExpr("firstLine(message)");
        expect(evalExpr(compiled, { message: "single line" }, buildEnv())).toBe("single line");
      });

      it("returns empty string for null inputs", () => {
        const compiled = compileExpr("firstLine(value)");
        expect(evalExpr(compiled, { value: null }, buildEnv())).toBe("");
      });

      it("composes with another builtin that takes a string", () => {
        const compiled = compileExpr("upper(firstLine(message))");
        expect(evalExpr(compiled, { message: "hello\nworld" }, buildEnv())).toBe("HELLO");
      });
    });

    describe("substring", () => {
      it("returns the first N characters when end is supplied", () => {
        const compiled = compileExpr("substring(message, 0, 5)");
        expect(evalExpr(compiled, { message: "hello world" }, buildEnv())).toBe("hello");
      });

      it("returns the tail when only start is supplied", () => {
        const compiled = compileExpr("substring(message, 6)");
        expect(evalExpr(compiled, { message: "hello world" }, buildEnv())).toBe("world");
      });

      it("clamps end past the string length", () => {
        const compiled = compileExpr("substring(message, 0, 999)");
        expect(evalExpr(compiled, { message: "hi" }, buildEnv())).toBe("hi");
      });

      it("returns empty string when end <= start", () => {
        const compiled = compileExpr("substring(message, 5, 2)");
        expect(evalExpr(compiled, { message: "hello world" }, buildEnv())).toBe("");
      });

      it("treats negative start as offset from the end", () => {
        const compiled = compileExpr("substring(message, -3)");
        expect(evalExpr(compiled, { message: "hello" }, buildEnv())).toBe("llo");
      });

      it("coerces non-string input via String(value)", () => {
        const compiled = compileExpr("substring(value, 0, 3)");
        expect(evalExpr(compiled, { value: 12345 }, buildEnv())).toBe("123");
      });

      it("returns empty string for null / undefined input", () => {
        const compiled = compileExpr("substring(value, 0, 5)");
        expect(evalExpr(compiled, { value: null }, buildEnv())).toBe("");
      });
    });

    describe("truncate", () => {
      it("returns the input unchanged when shorter than the limit", () => {
        const compiled = compileExpr('truncate(message, 80, "…")');
        expect(evalExpr(compiled, { message: "short" }, buildEnv())).toBe("short");
      });

      it("clips long input to N characters and appends the suffix", () => {
        const compiled = compileExpr('truncate(message, 5, "…")');
        expect(evalExpr(compiled, { message: "hello world" }, buildEnv())).toBe("hello…");
      });

      it("omits the suffix when none is supplied", () => {
        const compiled = compileExpr("truncate(message, 5)");
        expect(evalExpr(compiled, { message: "hello world" }, buildEnv())).toBe("hello");
      });

      it("returns the original text for non-numeric or negative limits", () => {
        const env = buildEnv();
        expect(evalExpr(compileExpr("truncate(message, value)"), { message: "abcdef", value: "nope" }, env)).toBe(
          "abcdef",
        );
        expect(evalExpr(compileExpr("truncate(message, -1)"), { message: "abcdef" }, env)).toBe("abcdef");
      });
    });

    describe("splitIndex", () => {
      it("returns the nth segment as a scalar", () => {
        // cel-js copies string literals verbatim — `"\n"` in CEL source is
        // backslash + n, not a newline — so `splitIndex` unescapes the
        // separator. The JS-side `\\n` here matches what an author would
        // type in their YAML cell expression.
        const compiled = compileExpr('splitIndex(message, "\\n", 0)');
        expect(evalExpr(compiled, { message: "first\nsecond\nthird" }, buildEnv())).toBe("first");
      });

      it("treats CRLF and bare CR the same as LF for a newline separator", () => {
        // Run output often arrives with Windows (`\r\n`) or classic-Mac (`\r`)
        // line endings. Splitting on a bare `\n` would leave a trailing `\r`
        // on the first segment, disagreeing with `firstLine`. They must match.
        const compiled = compileExpr('splitIndex(message, "\\n", 0)');
        expect(evalExpr(compiled, { message: "first\r\nsecond" }, buildEnv())).toBe("first");
        expect(evalExpr(compiled, { message: "first\rsecond" }, buildEnv())).toBe("first");
      });

      it("preserves a literal backslash that is not part of a recognized escape", () => {
        const compiled = compileExpr('splitIndex(value, "\\?", 0)');
        expect(evalExpr(compiled, { value: "a\\?b" }, buildEnv())).toBe("a");
      });

      it("supports non-newline separators", () => {
        const compiled = compileExpr('splitIndex(value, ",", 1)');
        expect(evalExpr(compiled, { value: "a,b,c" }, buildEnv())).toBe("b");
      });

      it("treats negative indexes as offsets from the end", () => {
        const compiled = compileExpr('splitIndex(value, ",", -1)');
        expect(evalExpr(compiled, { value: "a,b,c" }, buildEnv())).toBe("c");
      });

      it("returns empty string for out-of-range indexes", () => {
        const compiled = compileExpr('splitIndex(value, ",", 9)');
        expect(evalExpr(compiled, { value: "a,b,c" }, buildEnv())).toBe("");
      });

      it("returns the input unchanged when the separator is empty", () => {
        const compiled = compileExpr('splitIndex(value, "", 0)');
        expect(evalExpr(compiled, { value: "abc" }, buildEnv())).toBe("abc");
      });
    });

    describe("trim / replace / indexOf", () => {
      it("trims leading and trailing whitespace by default", () => {
        const compiled = compileExpr("trim(value)");
        expect(evalExpr(compiled, { value: "  hello  " }, buildEnv())).toBe("hello");
      });

      it("trims a custom character set when supplied", () => {
        const compiled = compileExpr('trim(value, "/")');
        expect(evalExpr(compiled, { value: "///path///" }, buildEnv())).toBe("path");
      });

      it("replaces every occurrence of old with new", () => {
        const compiled = compileExpr('replace(value, "a", "X")');
        expect(evalExpr(compiled, { value: "banana" }, buildEnv())).toBe("bXnXnX");
      });

      it("returns -1 from indexOf when the substring is missing", () => {
        const compiled = compileExpr('indexOf(value, "z")');
        expect(evalExpr(compiled, { value: "abc" }, buildEnv())).toBe(-1);
      });

      it("composes inside templated interpolation for runs-style trimming", () => {
        const template = compileTemplate('Summary: {{ truncate(firstLine(message), 20, "…") }}');
        const result = evalTemplate(template, { message: "  long\nrest of message" }, buildEnv(), String);
        expect(result).toBe("Summary:   long");
      });
    });

    describe("initial / firstInitial / githubAvatarOrInitial", () => {
      it("returns the first alphanumeric letter uppercased", () => {
        const compiled = compileExpr('initial("cloud-robot")');
        expect(evalExpr(compiled, {}, buildEnv())).toBe("C");
      });

      it("returns empty string for missing values", () => {
        const compiled = compileExpr("initial(value)");
        expect(evalExpr(compiled, { value: null }, buildEnv())).toBe("");
      });

      it("walks fallback values until it finds a usable initial", () => {
        const compiled = compileExpr('firstInitial("", " ", "", "Pedro Leão")');
        expect(evalExpr(compiled, {}, buildEnv())).toBe("P");
      });

      it("renders a github avatar when author.username is present", () => {
        const compiled = compileExpr("githubAvatarOrInitial(author, committer)");
        const out = evalExpr(
          compiled,
          {
            author: { name: "Pedro Leão", username: "forestileao" },
            committer: { name: "Pedro Leão" },
          },
          buildEnv(),
        );
        expect(out).toContain('src="https://github.com/forestileao.png"');
      });

      it("renders an initial badge when author.username is missing", () => {
        const compiled = compileExpr("githubAvatarOrInitial(author, committer)");
        const out = evalExpr(
          compiled,
          {
            author: { name: "cloud-robot" },
            committer: { name: "cloud-robot" },
          },
          buildEnv(),
        );
        expect(out).toBe('<div class="avatar avatar-fallback">C</div>');
      });
    });
  });

  describe("join builtin", () => {
    // `join` exists specifically so authors can flatten the result of a `.map`
    // macro chain into a single string. cel-js doesn't allow `.method()`
    // postfix after a function-call result, so the canonical form is
    // `join(list.map(x, expr), sep)`.
    it("joins a list of strings with the separator", () => {
      const compiled = compileExpr('join(["a", "b", "c"], ", ")');
      expect(evalExpr(compiled, {}, buildEnv())).toBe("a, b, c");
    });

    it("uses an empty separator when sep is missing or not a string", () => {
      const compiled = compileExpr('join(["x", "y"], 0)');
      expect(evalExpr(compiled, {}, buildEnv())).toBe("xy");
    });

    it("renders null / undefined / number elements as their string form", () => {
      const compiled = compileExpr('join(value, "|")');
      expect(evalExpr(compiled, { value: ["a", null, 2, "b"] }, buildEnv())).toBe("a||2|b");
    });

    it("returns an empty string for non-array inputs", () => {
      const compiled = compileExpr('join(value, ",")');
      expect(evalExpr(compiled, { value: "not a list" }, buildEnv())).toBe("");
      expect(evalExpr(compiled, { value: null }, buildEnv())).toBe("");
      expect(evalExpr(compiled, { value: 42 }, buildEnv())).toBe("");
    });

    it("composes with a list.map macro to render an HTML list", () => {
      const compiled = compileExpr('join(items.map(x, "<li>" + x.name + "</li>"), "")');
      const result = evalExpr(compiled, { items: [{ name: "alice" }, { name: "bob" }] }, buildEnv());
      expect(result).toBe("<li>alice</li><li>bob</li>");
    });

    it("composes with list.filter to skip rows before joining", () => {
      const compiled = compileExpr('join(items.filter(x, x.active).map(x, x.name), ", ")');
      const result = evalExpr(
        compiled,
        {
          items: [
            { name: "a", active: true },
            { name: "b", active: false },
            { name: "c", active: true },
          ],
        },
        buildEnv(),
      );
      expect(result).toBe("a, c");
    });

    it("works inside templated interpolation", () => {
      const template = compileTemplate('Hits: {{ join(tags, ", ") }}');
      const result = evalTemplate(template, { tags: ["red", "blue"] }, buildEnv(), String);
      expect(result).toBe("Hits: red, blue");
    });
  });

  describe("error reporting", () => {
    it("reports a compile error for invalid CEL", () => {
      const compiled = compileExpr("value /");
      const result = evalExprDetailed(compiled, {}, buildEnv());
      expect(result.ok).toBe(false);
      if (!result.ok) expect(result.error).toBeTruthy();
    });

    it("reports a type error when arithmetic cannot be coerced", () => {
      const compiled = compileExpr("value / 2");
      const result = evalExprDetailed(compiled, { value: "abc" }, buildEnv());
      expect(result.ok).toBe(false);
      if (!result.ok) expect(result.error).toMatch(/division|type/i);
    });

    it("propagates the first segment error from evalTemplateDetailed", () => {
      const template = compileTemplate("ok={{ value }} bad={{ value / }}");
      const result = evalTemplateDetailed(template, { value: 1 }, buildEnv(), String);
      expect(result.ok).toBe(false);
    });

    it("returns the rendered string when all segments succeed", () => {
      const template = compileTemplate("x={{ value / 2 }}");
      const result = evalTemplateDetailed(template, { value: "10" }, buildEnv(), String);
      expect(result.ok).toBe(true);
      if (result.ok) expect(result.value).toBe("x=5");
    });
  });
});
