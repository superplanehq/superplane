import { describe, expect, it } from "bun:test";

import { sanitizeSkillCommandName, skillCommandForSave, validateSkillName } from "./useSkillEditorPage";

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

describe("skillCommandForSave", () => {
  it("keeps the stored command until the name changes", () => {
    expect(skillCommandForSave("Review copy", "review-copy", false)).toBe("review-copy");
    expect(skillCommandForSave("Review copy", "review-copy", true)).toBe("reviewcopy");
  });
});
