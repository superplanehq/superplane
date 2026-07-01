import { describe, expect, it } from "vitest";

import { sanitizeHtml } from "./htmlSanitize";

/**
 * The sanitizer runs in jsdom under vitest. Each test exercises one specific
 * threat or allow-list rule documented in `htmlSanitize.ts`. Failures here
 * indicate a regression in the widget's security policy.
 */
const ROOT_ID = "html-root";

/**
 * Parse a sanitized HTML fragment for querying. The fragment is placed
 * directly into the body (no wrapper element) so `querySelector("div")`
 * matches the author's first `<div>`, not a wrapper introduced by the test.
 */
function parse(html: string): Document {
  return new DOMParser().parseFromString(html, "text/html");
}

describe("sanitizeHtml - dangerous elements", () => {
  it("strips <script> tags entirely", () => {
    const out = sanitizeHtml("Hi <script>window.x = 1</script> there", ROOT_ID);
    expect(out).not.toContain("<script");
    expect(out).toContain("Hi");
    expect(out).toContain("there");
  });

  it("strips <iframe>, <object>, <embed>", () => {
    const out = sanitizeHtml(
      '<iframe src="https://evil"></iframe><object data="x"></object><embed src="x"></embed>',
      ROOT_ID,
    );
    expect(out).not.toMatch(/<iframe|<object|<embed/);
  });

  it("strips head-like elements (<link>, <meta>, <base>, <title>)", () => {
    const out = sanitizeHtml(
      '<link rel="stylesheet" href="x.css"><meta name="csrf" content="x"><base href="x"><title>nope</title>',
      ROOT_ID,
    );
    expect(out).not.toMatch(/<link|<meta|<base|<title/);
  });

  it("strips form/input elements so authors cannot capture credentials", () => {
    const out = sanitizeHtml('<form action="x"><input name="pwd"><button>go</button></form>', ROOT_ID);
    expect(out).not.toMatch(/<form|<input|<button/);
  });

  it("strips media elements that could fetch external resources", () => {
    const out = sanitizeHtml(
      '<audio src="x.mp3"></audio><video src="x.mp4"></video><source src="x"><track src="x">',
      ROOT_ID,
    );
    expect(out).not.toMatch(/<audio|<video|<source|<track/);
  });
});

describe("sanitizeHtml - dangerous attributes", () => {
  it("removes inline event handlers from allowed tags", () => {
    const out = sanitizeHtml('<a href="https://example.com" onclick="alert(1)">link</a>', ROOT_ID);
    const doc = parse(out);
    const a = doc.querySelector("a");
    expect(a).not.toBeNull();
    expect(a!.getAttribute("onclick")).toBeNull();
  });

  it("rejects javascript: hrefs", () => {
    const out = sanitizeHtml('<a href="javascript:alert(1)">x</a>', ROOT_ID);
    const doc = parse(out);
    const a = doc.querySelector("a");
    // DOMPurify keeps the element but strips the dangerous href.
    expect(a?.getAttribute("href")).toBeNull();
  });

  it("rejects data: hrefs (could embed payloads)", () => {
    const out = sanitizeHtml('<a href="data:text/html,<script>1</script>">x</a>', ROOT_ID);
    const doc = parse(out);
    const a = doc.querySelector("a");
    expect(a?.getAttribute("href")).toBeNull();
  });

  it("allows http/https/mailto/tel hrefs", () => {
    const out = sanitizeHtml(
      '<a href="https://example.com">a</a><a href="mailto:hi@x">b</a><a href="tel:+1">c</a>',
      ROOT_ID,
    );
    const doc = parse(out);
    const links = doc.querySelectorAll("a");
    expect(links).toHaveLength(3);
    expect(links[0].getAttribute("href")).toBe("https://example.com");
    expect(links[1].getAttribute("href")).toBe("mailto:hi@x");
    expect(links[2].getAttribute("href")).toBe("tel:+1");
  });

  it("rejects protocol-relative hrefs that target arbitrary hosts", () => {
    const out = sanitizeHtml('<a href="//evil.example/phish">a</a><a href="/\\evil.example/phish">b</a>', ROOT_ID);
    const links = parse(out).querySelectorAll("a");
    expect(links).toHaveLength(2);
    expect(links[0].getAttribute("href")).toBeNull();
    expect(links[1].getAttribute("href")).toBeNull();
  });

  it("allows same-origin absolute and relative path hrefs", () => {
    const out = sanitizeHtml(
      '<a href="/dashboard">a</a><a href="./page">b</a><a href="../up">c</a><a href="#frag">d</a>',
      ROOT_ID,
    );
    const links = parse(out).querySelectorAll("a");
    expect(links).toHaveLength(4);
    expect(links[0].getAttribute("href")).toBe("/dashboard");
    expect(links[1].getAttribute("href")).toBe("./page");
    expect(links[2].getAttribute("href")).toBe("../up");
    expect(links[3].getAttribute("href")).toBe("#frag");
  });

  it("allows bare relative path hrefs without a scheme", () => {
    const out = sanitizeHtml(
      '<a href="logo.png">a</a><a href="page.html">b</a><a href="assets/img/logo.png?v=2">c</a>',
      ROOT_ID,
    );
    const links = parse(out).querySelectorAll("a");
    expect(links).toHaveLength(3);
    expect(links[0].getAttribute("href")).toBe("logo.png");
    expect(links[1].getAttribute("href")).toBe("page.html");
    expect(links[2].getAttribute("href")).toBe("assets/img/logo.png?v=2");
  });

  it("allows bare relative src/srcset on <img>", () => {
    const out = sanitizeHtml('<img src="logo.png" srcset="logo.png 1x, logo@2x.png 2x" alt="x">', ROOT_ID);
    const img = parse(out).querySelector("img");
    expect(img).not.toBeNull();
    expect(img!.getAttribute("src")).toBe("logo.png");
    expect(img!.getAttribute("srcset")).toBe("logo.png 1x, logo@2x.png 2x");
  });

  it("rejects unknown-scheme hrefs even without a slash (e.g. vbscript:)", () => {
    const out = sanitizeHtml('<a href="vbscript:msgbox(1)">a</a><a href="foo:bar">b</a>', ROOT_ID);
    const links = parse(out).querySelectorAll("a");
    expect(links).toHaveLength(2);
    expect(links[0].getAttribute("href")).toBeNull();
    expect(links[1].getAttribute("href")).toBeNull();
  });

  it("rejects protocol-relative src on <img>", () => {
    const out = sanitizeHtml('<img src="//evil.example/pixel.png" alt="a">', ROOT_ID);
    const img = parse(out).querySelector("img");
    expect(img).not.toBeNull();
    expect(img!.getAttribute("src")).toBeNull();
  });

  it("allows http(s) src/srcset on <img> but strips deprecated resource attrs", () => {
    // `src` and `srcset` survive when they point at http(s) so authors can
    // actually use <img>. `background` (deprecated <body> attr) and other
    // resource hooks for forbidden elements stay stripped as defense in depth.
    const out = sanitizeHtml(
      '<img src="https://cdn.example.com/x.png" srcset="https://cdn.example.com/y.png 2x" alt="x">' +
        '<div background="https://evil/bg.png">x</div>',
      ROOT_ID,
    );
    const doc = parse(out);
    const img = doc.querySelector("img");
    expect(img).not.toBeNull();
    expect(img!.getAttribute("src")).toBe("https://cdn.example.com/x.png");
    expect(img!.getAttribute("srcset")).toBe("https://cdn.example.com/y.png 2x");
    expect(img!.getAttribute("alt")).toBe("x");
    expect(doc.querySelector("div")?.getAttribute("background")).toBeNull();
  });

  it("rejects javascript: and data: src on <img>", () => {
    const out = sanitizeHtml(
      '<img src="javascript:alert(1)" alt="a"><img src="data:text/html,<script>1</script>" alt="b">',
      ROOT_ID,
    );
    const imgs = parse(out).querySelectorAll("img");
    expect(imgs).toHaveLength(2);
    expect(imgs[0].getAttribute("src")).toBeNull();
    expect(imgs[1].getAttribute("src")).toBeNull();
  });

  it("strips inline style values that load external resources", () => {
    const out = sanitizeHtml('<div style="background-image: url(https://evil/x.png); color: red;">x</div>', ROOT_ID);
    const doc = parse(out);
    // The dangerous style attribute is dropped wholesale rather than partly
    // sanitized, so authors cannot smuggle one bad rule alongside good ones.
    expect(doc.querySelector("div")?.getAttribute("style")).toBeNull();
  });

  it("allows safe inline style attributes", () => {
    const out = sanitizeHtml('<p style="color: red; padding: 4px;">x</p>', ROOT_ID);
    const doc = parse(out);
    expect(doc.querySelector("p")?.getAttribute("style")).toMatch(/color:\s*red/);
  });

  it("preserves class attribute (tailwind classes)", () => {
    const out = sanitizeHtml('<div class="flex gap-2 text-slate-800">x</div>', ROOT_ID);
    const doc = parse(out);
    expect(doc.querySelector("div")?.getAttribute("class")).toBe("flex gap-2 text-slate-800");
  });

  it("preserves aria-* attributes for accessible markup", () => {
    const out = sanitizeHtml(
      '<div role="alert" aria-label="status" aria-hidden="true"><span aria-describedby="foo">x</span></div>',
      ROOT_ID,
    );
    const doc = parse(out);
    const div = doc.querySelector("div");
    expect(div?.getAttribute("role")).toBe("alert");
    expect(div?.getAttribute("aria-label")).toBe("status");
    expect(div?.getAttribute("aria-hidden")).toBe("true");
    expect(doc.querySelector("span")?.getAttribute("aria-describedby")).toBe("foo");
  });

  it("drops data-* attributes (only aria-* is allowed beyond the allow-list)", () => {
    const out = sanitizeHtml('<div data-secret="x" aria-label="ok">y</div>', ROOT_ID);
    const div = parse(out).querySelector("div");
    expect(div?.getAttribute("data-secret")).toBeNull();
    expect(div?.getAttribute("aria-label")).toBe("ok");
  });

  it("drops disallowed attributes (e.g. style on a removed tag)", () => {
    const out = sanitizeHtml('<script style="x"></script>', ROOT_ID);
    expect(out).not.toContain("script");
    expect(out).not.toContain("style");
  });
});

describe("sanitizeHtml - <style> blocks", () => {
  it("scopes selectors to the widget root", () => {
    const out = sanitizeHtml("<style>p { color: red; } .foo, h2 { font-weight: bold; }</style><p>x</p>", ROOT_ID);
    const doc = parse(out);
    const style = doc.querySelector("style");
    expect(style).not.toBeNull();
    const css = style!.textContent ?? "";
    // Every selector is prefixed; the comma-separated list keeps each piece scoped.
    expect(css).toMatch(/\[data-console-html-root="html-root"\]\s+p\s*\{/);
    expect(css).toMatch(/\[data-console-html-root="html-root"\]\s+\.foo/);
    expect(css).toMatch(/\[data-console-html-root="html-root"\]\s+h2/);
    // Unscoped versions are gone.
    expect(css).not.toMatch(/(^|\})\s*p\s*\{/);
  });

  it("drops @import even when nested inside other content", () => {
    const out = sanitizeHtml('<style>@import "https://evil/x.css"; p { color: red; }</style>', ROOT_ID);
    const doc = parse(out);
    const css = doc.querySelector("style")?.textContent ?? "";
    expect(css).not.toMatch(/@import/i);
    expect(css).toMatch(/color:\s*red/);
  });

  it("drops rules that reference url() to block external resource fetches", () => {
    const out = sanitizeHtml(
      "<style>.bad { background: url(https://evil/x.png); } .ok { color: blue; }</style>",
      ROOT_ID,
    );
    const css = parse(out).querySelector("style")?.textContent ?? "";
    expect(css).not.toContain("url(");
    expect(css).not.toMatch(/\.bad/);
    expect(css).toMatch(/\.ok/);
    expect(css).toMatch(/color:\s*blue/);
  });

  it("keeps selectors with commas inside :not()/:is() intact when scoping", () => {
    const out = sanitizeHtml("<style>div:not(.a, .b) { color: red; }</style>", ROOT_ID);
    const css = parse(out).querySelector("style")?.textContent ?? "";
    // The whole selector is scoped once; the inner list is not split apart.
    expect(css).toMatch(/\[data-console-html-root="html-root"\]\s+div:not\(\.a, \.b\)\s*\{/);
    expect(css).toMatch(/color:\s*red/);
  });

  it("splits a real top-level selector list while preserving nested commas", () => {
    const out = sanitizeHtml("<style>:is(h1, h2), p:not(.x, .y) { font-weight: bold; }</style>", ROOT_ID);
    const css = parse(out).querySelector("style")?.textContent ?? "";
    expect(css).toMatch(/\[data-console-html-root="html-root"\]\s+:is\(h1, h2\)/);
    expect(css).toMatch(/\[data-console-html-root="html-root"\]\s+p:not\(\.x, \.y\)/);
  });

  it("does not split on commas inside attribute selector values", () => {
    const out = sanitizeHtml('<style>[data-x="a,b"] { color: blue; }</style>', ROOT_ID);
    const css = parse(out).querySelector("style")?.textContent ?? "";
    expect(css).toMatch(/\[data-console-html-root="html-root"\]\s+\[data-x="a,b"\]\s*\{/);
  });

  it("scopes rules nested inside @media", () => {
    const out = sanitizeHtml("<style>@media (min-width: 600px) { p { color: green; } }</style>", ROOT_ID);
    const css = parse(out).querySelector("style")?.textContent ?? "";
    expect(css).toMatch(/@media/);
    expect(css).toMatch(/\[data-console-html-root="html-root"\]\s+p/);
  });
});

describe("sanitizeHtml - allowed content", () => {
  it("keeps a typical HTML widget body intact", () => {
    const out = sanitizeHtml(
      '<section class="flex flex-col gap-2"><h2>Status</h2><p>All clear.</p></section>',
      ROOT_ID,
    );
    const doc = parse(out);
    expect(doc.querySelector("section")?.getAttribute("class")).toBe("flex flex-col gap-2");
    expect(doc.querySelector("h2")?.textContent).toBe("Status");
    expect(doc.querySelector("p")?.textContent).toBe("All clear.");
  });

  it("keeps <details>/<summary> with the open attribute", () => {
    const out = sanitizeHtml("<details open><summary>S</summary><p>body</p></details>", ROOT_ID);
    const doc = parse(out);
    const details = doc.querySelector("details");
    expect(details).not.toBeNull();
    expect(details!.hasAttribute("open")).toBe(true);
    expect(doc.querySelector("summary")?.textContent).toBe("S");
  });

  it("returns empty string for empty input", () => {
    expect(sanitizeHtml("", ROOT_ID)).toBe("");
  });
});
