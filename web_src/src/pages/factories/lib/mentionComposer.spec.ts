import { describe, expect, it } from "vitest";
import { buildMentionToken, detectMentionTrigger, extractMentionedUserIds } from "./mentionComposer";

describe("detectMentionTrigger", () => {
  it("is inactive on empty text", () => {
    expect(detectMentionTrigger("", 0)).toEqual({ active: false, query: "", start: -1 });
  });

  it("activates for an @ at the start of the text", () => {
    expect(detectMentionTrigger("@ali", 4)).toEqual({ active: true, query: "ali", start: 0 });
  });

  it("activates for an @ preceded by whitespace", () => {
    expect(detectMentionTrigger("hey @ali", 8)).toEqual({ active: true, query: "ali", start: 4 });
  });

  it("does not activate for an @ in the middle of a word (e.g. an email)", () => {
    expect(detectMentionTrigger("me@example.com", 14)).toEqual({ active: false, query: "", start: -1 });
  });

  it("only looks at text up to the cursor, not the whole value", () => {
    // Cursor sits right after "hey ", before the @ — nothing to trigger yet.
    expect(detectMentionTrigger("hey @ali", 4)).toEqual({ active: false, query: "", start: -1 });
  });

  it("does not trigger across a newline", () => {
    expect(detectMentionTrigger("@ali\nnew line", 4)).toEqual({ active: true, query: "ali", start: 0 });
    expect(detectMentionTrigger("@ali\nnew line", 13)).toEqual({ active: false, query: "", start: -1 });
  });

  it("does not reopen the picker inside an already-inserted mention token", () => {
    const body = "@[Alice](user:11111111-1111-1111-1111-111111111111) thanks!";
    // Cursor placed inside the token, e.g. right after "Alice".
    const cursor = body.indexOf("Alice") + "Alice".length;
    expect(detectMentionTrigger(body, cursor)).toEqual({ active: false, query: "", start: -1 });
  });

  it("allows spaces in the query so multi-word names can be matched", () => {
    expect(detectMentionTrigger("@Jamie Op", 9)).toEqual({ active: true, query: "Jamie Op", start: 0 });
  });
});

describe("buildMentionToken", () => {
  it("builds a @[Name](user:id) token", () => {
    expect(buildMentionToken("Ada Lovelace", "user-1")).toBe("@[Ada Lovelace](user:user-1)");
  });

  it("strips markdown link syntax from the name defensively", () => {
    expect(buildMentionToken("Weird](Name)", "user-1")).toBe("@[WeirdName](user:user-1)");
  });
});

describe("extractMentionedUserIds", () => {
  const idA = "11111111-1111-1111-1111-111111111111";
  const idB = "22222222-2222-2222-2222-222222222222";

  it("returns an empty list for plain text", () => {
    expect(extractMentionedUserIds("no mentions here")).toEqual([]);
  });

  it("extracts a single mentioned id", () => {
    expect(extractMentionedUserIds(`Hey @[Alice](user:${idA}), take a look`)).toEqual([idA]);
  });

  it("extracts multiple mentioned ids in document order", () => {
    expect(extractMentionedUserIds(`@[Bob](user:${idB}) cc @[Alice](user:${idA})`)).toEqual([idB, idA]);
  });

  it("dedupes repeated mentions of the same user", () => {
    expect(extractMentionedUserIds(`@[Alice](user:${idA}) again @[Alice](user:${idA})`)).toEqual([idA]);
  });

  it("drops a token whose markup was broken by editing (no closing paren)", () => {
    expect(extractMentionedUserIds(`@[Alice](user:${idA}`)).toEqual([]);
  });

  it("ignores non-uuid content in a user: link", () => {
    expect(extractMentionedUserIds("@[Alice](user:not-a-uuid)")).toEqual([]);
  });
});
