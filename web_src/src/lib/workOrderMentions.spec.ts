import { describe, expect, it } from "vitest";

import {
  filterMentionCandidates,
  insertMentionAtCursor,
  mentionQueryAtCursor,
  retainMentions,
} from "./workOrderMentions";

const alice = { id: "alice", name: "Alice Anderson", email: "alice@example.com" };
const bob = { id: "bob", name: "Bob Brown", email: "bob@example.com" };

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
});
