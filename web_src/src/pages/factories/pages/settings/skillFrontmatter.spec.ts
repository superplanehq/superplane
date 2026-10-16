import { describe, expect, it } from "bun:test";

import {
  sanitizeSkillCommandName,
  setSkillFrontmatterFields,
  skillDisplayTitle,
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

  it("drops an inline comment and keeps a hash inside quotes", () => {
    expect(skillFrontmatterField("---\nname: review-copy # command name\n---\n", "name")).toBe("review-copy");
    expect(skillFrontmatterField('---\ntitle: "review # copy" # label\n---\n', "title")).toBe("review # copy");
  });

  it("keeps escaped quotes inside a double-quoted title", () => {
    expect(skillFrontmatterField('---\ntitle: "Review \\"copy\\"" # label\n---\n', "title")).toBe('Review "copy"');
  });

  it("keeps doubled quotes inside a single-quoted title", () => {
    expect(skillFrontmatterField("---\ntitle: 'Review ''copy'''\n---\n", "title")).toBe("Review 'copy'");
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

  it("round-trips a title that contains quotes and a backslash", () => {
    const title = 'Review "copy" \\ path';
    const next = setSkillFrontmatterFields("---\ntitle: \n---\n", { title });
    expect(skillFrontmatterField(next, "title")).toBe(title);
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
