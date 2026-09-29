#!/usr/bin/env bun
// Flatten nested JUnit testsuites into one suite per file.
//
// Semaphore's generic test-results parser reads only testcase children of
// top-level testsuites. Bun nests describe() blocks as extra testsuites, so
// published reports show zero tests unless this script runs first.

import fs from "node:fs";
import process from "node:process";

function decode(value) {
  return value
    .replaceAll("&lt;", "<")
    .replaceAll("&gt;", ">")
    .replaceAll("&quot;", '"')
    .replaceAll("&apos;", "'")
    .replaceAll("&amp;", "&");
}

function encode(value) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function parseAttributes(source) {
  const attrs = {};
  for (const match of source.matchAll(/([^\s=]+)="([^"]*)"/g)) {
    attrs[match[1]] = decode(match[2]);
  }
  return attrs;
}

function skipDeclarationAndSpace(xml, index) {
  let i = index;
  while (i < xml.length) {
    while (i < xml.length && /\s/.test(xml[i])) {
      i += 1;
    }
    if (xml.startsWith("<?", i)) {
      const end = xml.indexOf("?>", i);
      i = end === -1 ? xml.length : end + 2;
      continue;
    }
    if (xml.startsWith("<!--", i)) {
      const end = xml.indexOf("-->", i);
      i = end === -1 ? xml.length : end + 3;
      continue;
    }
    break;
  }
  return i;
}

function parseNode(xml, start) {
  let i = skipDeclarationAndSpace(xml, start);
  if (xml[i] !== "<" || xml.startsWith("</", i)) {
    throw new Error(`Expected an opening tag at index ${i}`);
  }

  const tagEnd = xml.indexOf(">", i);
  if (tagEnd === -1) {
    throw new Error("Unclosed tag");
  }

  const raw = xml.slice(i + 1, tagEnd);
  const selfClosing = raw.endsWith("/");
  const body = (selfClosing ? raw.slice(0, -1) : raw).trim();
  const space = body.search(/\s/);
  const tag = space === -1 ? body : body.slice(0, space);
  const node = {
    tag,
    attrs: parseAttributes(space === -1 ? "" : body.slice(space)),
    children: [],
    text: "",
  };
  i = tagEnd + 1;
  if (selfClosing) {
    return { node, end: i };
  }

  const close = `</${tag}>`;
  while (i < xml.length) {
    i = skipDeclarationAndSpace(xml, i);
    if (xml.startsWith(close, i)) {
      return { node, end: i + close.length };
    }
    if (xml[i] === "<") {
      const child = parseNode(xml, i);
      node.children.push(child.node);
      i = child.end;
      continue;
    }
    const next = xml.indexOf("<", i);
    if (next === -1) {
      throw new Error(`Missing close tag for <${tag}>`);
    }
    node.text += decode(xml.slice(i, next));
    i = next;
  }

  throw new Error(`Missing close tag for <${tag}>`);
}

function serialize(node, depth = 0) {
  const pad = "  ".repeat(depth);
  const attrs = Object.entries(node.attrs)
    .map(([name, value]) => ` ${name}="${encode(value)}"`)
    .join("");
  if (node.children.length === 0 && node.text === "") {
    return `${pad}<${node.tag}${attrs}/>`;
  }
  if (node.children.length === 0) {
    return `${pad}<${node.tag}${attrs}>${encode(node.text)}</${node.tag}>`;
  }
  const inner = node.children.map((child) => serialize(child, depth + 1)).join("\n");
  return `${pad}<${node.tag}${attrs}>\n${inner}\n${pad}</${node.tag}>`;
}

function collectCases(suite, describePath, cases) {
  for (const child of suite.children) {
    if (child.tag === "testsuite") {
      const name = child.attrs.name || "";
      const nestedPath = name ? [...describePath, name] : describePath;
      collectCases(child, nestedPath, cases);
      continue;
    }
    if (child.tag !== "testcase") {
      continue;
    }
    const testCase = structuredClone(child);
    if (describePath.length > 0) {
      testCase.attrs.name = [...describePath, testCase.attrs.name || ""].join(" > ");
    }
    cases.push(testCase);
  }
}

function flattenJunit(root) {
  if (root.tag !== "testsuites") {
    return;
  }

  for (const suite of root.children) {
    if (suite.tag !== "testsuite") {
      continue;
    }
    const cases = [];
    collectCases(suite, [], cases);
    suite.children = cases;
    suite.text = "";
    let failures = 0;
    let errors = 0;
    let skipped = 0;
    for (const testCase of cases) {
      if (testCase.children.some((child) => child.tag === "failure")) {
        failures += 1;
      }
      if (testCase.children.some((child) => child.tag === "error")) {
        errors += 1;
      }
      if (testCase.children.some((child) => child.tag === "skipped")) {
        skipped += 1;
      }
    }
    suite.attrs.tests = String(cases.length);
    suite.attrs.failures = String(failures);
    suite.attrs.errors = String(errors);
    suite.attrs.skipped = String(skipped);
  }
}

function main() {
  const junitFile = process.argv[2];
  if (!junitFile || !fs.existsSync(junitFile) || fs.statSync(junitFile).size === 0) {
    return;
  }

  const xml = fs.readFileSync(junitFile, "utf8");
  const { node: root } = parseNode(xml, 0);
  flattenJunit(root);
  fs.writeFileSync(junitFile, `<?xml version="1.0" encoding="UTF-8"?>\n${serialize(root)}\n`);
}

main();
