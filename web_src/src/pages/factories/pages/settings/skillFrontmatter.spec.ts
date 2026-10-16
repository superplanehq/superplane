import { describe, expect, it } from "bun:test";

import {
  sanitizeSkillCommandName,
  setSkillFrontmatterFields,
  skillDisplayTitle,
  skillFieldsFromMarkdown,
  skillFrontmatterField,
  skillListDescription,
} from "./skillFrontmatter";

const SAMPLE = `---
name: oypirate
title: Oy Pirate!
description: Talk like a pirate.
---

Ahoy.
`;

describe("sanitizeSkillCommandName", () => {
  it("removes spaces and punctuation from the typed name", () => {
    expect(sanitizeSkillCommandName("Oy Pirate!")).toBe("oypirate");
  });

  it("keeps lowercase letters, numbers, and dashes", () => {
    expect(sanitizeSkillCommandName("review-copy")).toBe("review-copy");
    expect(sanitizeSkillCommandName("Review Copy 2")).toBe("reviewcopy2");
  });

  it("drops leading digits and dashes so the command starts with a letter", () => {
    expect(sanitizeSkillCommandName("123abc")).toBe("abc");
    expect(sanitizeSkillCommandName("---hello")).toBe("hello");
  });

  it("returns an empty slug when nothing valid remains", () => {
    expect(sanitizeSkillCommandName("!!!")).toBe("");
    expect(sanitizeSkillCommandName("   ")).toBe("");
  });
});

describe("skillFrontmatterField", () => {
  it("reads name, title, and description", () => {
    expect(skillFrontmatterField(SAMPLE, "name")).toBe("oypirate");
    expect(skillFrontmatterField(SAMPLE, "title")).toBe("Oy Pirate!");
    expect(skillFrontmatterField(SAMPLE, "description")).toBe("Talk like a pirate.");
  });

  it("strips quotes around YAML scalars", () => {
    expect(skillFrontmatterField('---\ntitle: "Oy Pirate!"\n---\n', "title")).toBe("Oy Pirate!");
  });
});

describe("setSkillFrontmatterFields", () => {
  it("writes name and title into existing frontmatter", () => {
    const next = setSkillFrontmatterFields("---\nname: \ndescription: \n---\n\n", {
      name: "oypirate",
      title: "Oy Pirate!",
    });
    expect(skillFrontmatterField(next, "name")).toBe("oypirate");
    expect(skillFrontmatterField(next, "title")).toBe("Oy Pirate!");
    expect(skillFrontmatterField(next, "description")).toBe("");
  });
});

describe("skillFieldsFromMarkdown", () => {
  it("uses name for both the title and the command when only name is set", () => {
    expect(skillFieldsFromMarkdown("---\nname: ponytail-review\n---\n")).toEqual({
      title: "ponytail-review",
      command: "ponytail-review",
    });
  });

  it("uses title for the Name field and derives the command from it when only title is set", () => {
    expect(skillFieldsFromMarkdown('---\ntitle: "Oy Pirate!"\n---\n')).toEqual({
      title: "Oy Pirate!",
      command: "oypirate",
    });
  });

  it("prefers title for the Name field and name for the command when both are set", () => {
    expect(skillFieldsFromMarkdown("---\nname: oypirate\ntitle: Oy Pirate Pro\n---\n")).toEqual({
      title: "Oy Pirate Pro",
      command: "oypirate",
    });
  });

  it("returns undefined values when name and title are empty", () => {
    expect(skillFieldsFromMarkdown("---\nname: \ntitle: \ndescription: \n---\n\n")).toEqual({
      title: undefined,
      command: undefined,
    });
  });

  it("ignores inline YAML comments after plain and quoted values", () => {
    expect(skillFieldsFromMarkdown("---\nname: review-copy # team skill\n---\n")).toEqual({
      title: "review-copy",
      command: "review-copy",
    });
    expect(skillFieldsFromMarkdown('---\nname: review-copy\ntitle: "Review copy" # team skill\n---\n')).toEqual({
      title: "Review copy",
      command: "review-copy",
    });
  });

  it("returns undefined values when there is no frontmatter", () => {
    expect(skillFieldsFromMarkdown("Ahoy.")).toEqual({ title: undefined, command: undefined });
  });

  it("sanitizes invalid characters in the command but keeps the title as written", () => {
    expect(skillFieldsFromMarkdown("---\nname: Ponytail Review!\n---\n")).toEqual({
      title: "Ponytail Review!",
      command: "ponytailreview",
    });
  });
});

describe("skillDisplayTitle", () => {
  it("prefers the YAML title over the slug", () => {
    expect(skillDisplayTitle({ name: "oypirate", markdown: SAMPLE })).toBe("Oy Pirate!");
  });

  it("falls back to the resource name", () => {
    expect(skillDisplayTitle({ name: "review-copy", markdown: "" })).toBe("review-copy");
  });
});

describe("skillListDescription", () => {
  it("reads the YAML description", () => {
    expect(skillListDescription({ markdown: SAMPLE })).toBe("Talk like a pirate.");
  });

  it("falls back to a GitHub ref when markdown has no description", () => {
    expect(
      skillListDescription({
        repository: "acme/skill",
        ref: "v1",
      }),
    ).toBe("acme/skill@v1");
  });
});
