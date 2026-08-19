import { describe, expect, it } from "vitest";

import {
  filterMentionCandidates,
  insertMentionAtCursor,
  mentionCandidateByName,
  mentionQueryAtCursor,
  mentionsInBody,
  retainMentions,
  splitMentionSegments,
} from "./workOrderMentions";

const alice = { id: "alice", name: "Alice Anderson", email: "alice@example.com" };
const aliceShort = { id: "alice-short", name: "Alice", email: "alice-short@example.com" };
const bob = { id: "bob", name: "Bob", email: "bob@example.com" };
const bobby = { id: "bobby", name: "Bobby", email: "bobby@example.com" };

describe("mentionQueryAtCursor", () => {
  it("opens after @ at the start of the comment", () => {
    expect(mentionQueryAtCursor("@al", 3)).toEqual({ start: 0, query: "al" });
  });

  it("opens after @ that follows whitespace", () => {
    expect(mentionQueryAtCursor("Hey @bo", 7)).toEqual({ start: 4, query: "bo" });
  });

  it("does not treat an email address as a mention", () => {
    expect(mentionQueryAtCursor("write alice@example.com", 23)).toBeNull();
  });

  it("closes when a space follows the query", () => {
    expect(mentionQueryAtCursor("Hey @bo ", 8)).toBeNull();
  });
});

describe("filterMentionCandidates", () => {
  it("filters by name or email and caps the list", () => {
    expect(filterMentionCandidates([alice, bob], "ali")).toEqual([alice]);
    expect(filterMentionCandidates([alice, bob], "BOB@")).toEqual([bob]);
    expect(filterMentionCandidates([alice, bob], "")).toEqual([alice, bob]);
  });
});

describe("insertMentionAtCursor", () => {
  it("replaces the @query with @Name and a trailing space", () => {
    expect(insertMentionAtCursor("Hey @al", 7, "Alice Anderson")).toEqual({
      value: "Hey @Alice Anderson ",
      cursor: 20,
    });
  });
});

describe("retainMentions", () => {
  it("drops mentions whose @Name is no longer in the body", () => {
    expect(retainMentions([alice, bob], "Thanks @Alice Anderson")).toEqual([alice]);
  });

  it("does not keep a shorter name inside a longer @Name", () => {
    expect(retainMentions([bob, bobby], "Thanks @Bobby")).toEqual([bobby]);
  });

  it("keeps the longer name when both nested names are tracked", () => {
    expect(retainMentions([aliceShort, alice], "Thanks @Alice Anderson")).toEqual([alice]);
  });

  it("keeps the shorter name when only that mention is tracked", () => {
    expect(retainMentions([aliceShort], "Thanks @Alice Anderson")).toEqual([aliceShort]);
  });

  it("keeps one ID per @Name when two members share that name", () => {
    const aliceA = { id: "alice-a", name: "Alice Anderson" };
    const aliceB = { id: "alice-b", name: "Alice Anderson" };
    expect(retainMentions([aliceA, aliceB], "Thanks @Alice Anderson")).toEqual([aliceA]);
    expect(retainMentions([aliceA, aliceB], "Thanks @Alice Anderson and @Alice Anderson")).toEqual([aliceA, aliceB]);
  });
});

describe("mentionsInBody", () => {
  it("attaches a typed complete @Name even without picker tracking", () => {
    expect(mentionsInBody([alice, bob], "Thanks @Alice Anderson")).toEqual([alice]);
  });

  it("prefers the picker-selected ID when two members share a name", () => {
    const aliceA = { id: "alice-a", name: "Alice Anderson" };
    const aliceB = { id: "alice-b", name: "Alice Anderson" };
    expect(mentionsInBody([aliceA, aliceB], "Thanks @Alice Anderson", [aliceB])).toEqual([aliceB]);
  });
});

describe("splitMentionSegments", () => {
  it("wraps a complete @Name including spaces", () => {
    expect(splitMentionSegments("@test test dasdasd", ["test test"])).toEqual([
      { text: "@test test", mention: true },
      { text: " dasdasd", mention: false },
    ]);
  });

  it("keeps the longer name when a shorter name is a prefix", () => {
    expect(splitMentionSegments("Thanks @Alice Anderson", ["Alice", "Alice Anderson"])).toEqual([
      { text: "Thanks ", mention: false },
      { text: "@Alice Anderson", mention: true },
    ]);
  });

  it("does not highlight an incomplete query", () => {
    expect(splitMentionSegments("Hey @Ali", ["Alice Anderson"])).toEqual([{ text: "Hey @Ali", mention: false }]);
  });
});

describe("mentionCandidateByName", () => {
  it("finds the member whose name matches the @token", () => {
    expect(mentionCandidateByName([alice, bob], "@Alice Anderson")).toEqual(alice);
  });
});
