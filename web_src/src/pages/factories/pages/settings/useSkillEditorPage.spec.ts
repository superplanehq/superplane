import { describe, expect, it } from "bun:test";

import { sanitizeSkillCommandName, validateSkillName } from "./useSkillEditorPage";

describe("validateSkillName", () => {
  it("requires a command slug", () => {
    expect(validateSkillName("")).toBe("Name is required.");
    expect(validateSkillName("superplane")).toBe("The name superplane is reserved.");
    expect(validateSkillName("oypirate")).toBe("");
  });
});

describe("sanitizeSkillCommandName", () => {
  it("re-exports the frontmatter sanitizer", () => {
    expect(sanitizeSkillCommandName("Oy Pirate!")).toBe("oypirate");
  });
});
