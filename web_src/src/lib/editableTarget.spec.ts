import { describe, expect, it } from "vitest";

import { isEditableTarget } from "./editableTarget";

function makeElement(html: string): Element {
  const container = document.createElement("div");
  container.innerHTML = html;
  return container.firstElementChild as Element;
}

describe("isEditableTarget", () => {
  it("returns false for null and non-Element targets", () => {
    expect(isEditableTarget(null)).toBe(false);
    expect(isEditableTarget(document)).toBe(false);
  });

  it("returns false for a plain, non-editable element", () => {
    expect(isEditableTarget(makeElement("<div><span>x</span></div>"))).toBe(false);
  });

  it.each([
    ["<input />"],
    ["<textarea></textarea>"],
    ["<select></select>"],
    ['<div contenteditable="true"></div>'],
    ['<div class="monaco-editor"></div>'],
  ])("returns true for editable element %s", (html) => {
    expect(isEditableTarget(makeElement(html))).toBe(true);
  });

  it("returns true for a descendant of a Monaco editor", () => {
    const monaco = makeElement('<div class="monaco-editor"><span data-inner></span></div>');
    const inner = monaco.querySelector("[data-inner]") as Element;
    expect(isEditableTarget(inner)).toBe(true);
  });
});
