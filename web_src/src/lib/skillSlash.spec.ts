import { describe, expect, it } from "bun:test";

import {
  filterSkillSlashCandidates,
  insertSkillSlashAtCursor,
  skillSlashCandidatesFromResources,
  skillSlashQueryAtCursor,
} from "./skillSlash";

const CANDIDATES = [
  { id: "1", command: "oypirate", title: "Oy Pirate!", description: "Talk like a pirate." },
  { id: "2", command: "review-copy", title: "Review copy", description: "Write STE copy." },
];

describe("skillSlashQueryAtCursor", () => {
  it("detects a slash at the start of the input", () => {
    expect(skillSlashQueryAtCursor("/oy", 3)).toEqual({ start: 0, query: "oy" });
  });

  it("detects a slash after whitespace", () => {
    expect(skillSlashQueryAtCursor("use /oy", 7)).toEqual({ start: 4, query: "oy" });
  });

  it("ignores a slash in the middle of a token", () => {
    expect(skillSlashQueryAtCursor("https://example.com", 19)).toBeNull();
  });
});

describe("filterSkillSlashCandidates", () => {
  it("matches command, title, and description", () => {
    expect(filterSkillSlashCandidates(CANDIDATES, "oy").map((entry) => entry.command)).toEqual(["oypirate"]);
    expect(filterSkillSlashCandidates(CANDIDATES, "pirate").map((entry) => entry.command)).toEqual(["oypirate"]);
    expect(filterSkillSlashCandidates(CANDIDATES, "STE").map((entry) => entry.command)).toEqual(["review-copy"]);
  });
});

describe("insertSkillSlashAtCursor", () => {
  it("replaces the slash query with the command", () => {
    expect(insertSkillSlashAtCursor("/oy", 3, "oypirate")).toEqual({
      value: "/oypirate ",
      cursor: 10,
    });
  });
});

describe("skillSlashCandidatesFromResources", () => {
  it("skips disabled skills and empty names", () => {
    expect(
      skillSlashCandidatesFromResources([
        { id: "1", name: "oypirate", enabled: true, markdown: "---\ntitle: Oy Pirate!\ndescription: Ahoy.\n---\n" },
        { id: "2", name: "off", enabled: false },
        { id: "3", name: "", enabled: true },
      ]),
    ).toEqual([{ id: "1", command: "oypirate", title: "Oy Pirate!", description: "Ahoy." }]);
  });
});
