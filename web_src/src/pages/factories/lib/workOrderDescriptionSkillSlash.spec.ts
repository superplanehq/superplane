import { describe, expect, it, vi } from "bun:test";

import { skillSlashQueryAtCursor } from "@/lib/skillSlash";

import { applySkillSlashMenuKeyDown } from "./workOrderDescriptionSkillSlash";

describe("skill slash query in a description block", () => {
  it("matches a slash query in the current block", () => {
    expect(skillSlashQueryAtCursor("/oy", 3)).toEqual({ start: 0, query: "oy" });
    expect(skillSlashQueryAtCursor("use /oy", 7)).toEqual({ start: 4, query: "oy" });
  });

  it("closes after whitespace so Enter can create the task", () => {
    expect(skillSlashQueryAtCursor("/oypirate ", 10)).toBeNull();
  });
});

describe("applySkillSlashMenuKeyDown", () => {
  it("inserts on Enter and ignores Command+Enter", () => {
    const onSelect = vi.fn();
    const enter = { key: "Enter", metaKey: false, ctrlKey: false, preventDefault: vi.fn() } as unknown as KeyboardEvent;
    const commandEnter = {
      key: "Enter",
      metaKey: true,
      ctrlKey: false,
      preventDefault: vi.fn(),
    } as unknown as KeyboardEvent;
    const menu = {
      open: true,
      count: 1,
      highlightIndex: 0,
      onHighlight: vi.fn(),
      onSelect,
      onDismiss: vi.fn(),
    };

    expect(applySkillSlashMenuKeyDown(enter, menu)).toBe(true);
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(applySkillSlashMenuKeyDown(commandEnter, menu)).toBe(false);
  });
});
