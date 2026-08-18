import { describe, expect, it } from "vitest";

import {
  buildLatestArtifactDataById,
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  extractArtifactUrl,
  extractPrArtifactState,
  formatPrArtifactLabel,
  normalizeArtifactKind,
  overlayLiveArtifactData,
  resolveBranchArtifactUrl,
  toArtifactDataRecord,
  toWorkOrderArtifactLikes,
} from "./workOrderArtifact";

describe("formatPrArtifactLabel", () => {
  it("returns #<number> when data.number is set (backend convention)", () => {
    expect(formatPrArtifactLabel({ number: 1234 })).toBe("#1234");
    expect(formatPrArtifactLabel({ number: "1234" })).toBe("#1234");
  });

  it("falls back to data.prNumber when data.number is absent", () => {
    expect(formatPrArtifactLabel({ prNumber: 42 })).toBe("#42");
  });

  it("prefers data.number over data.prNumber when both are set", () => {
    expect(formatPrArtifactLabel({ number: 1, prNumber: 2 })).toBe("#1");
  });

  it("normalizes a leading '#' so callers don't render '##'", () => {
    expect(formatPrArtifactLabel({ number: "#42" })).toBe("#42");
    expect(formatPrArtifactLabel({ prNumber: "#7" })).toBe("#7");
  });

  it("returns undefined when no known key carries a usable value", () => {
    expect(formatPrArtifactLabel(undefined)).toBeUndefined();
    expect(formatPrArtifactLabel({})).toBeUndefined();
    expect(formatPrArtifactLabel({ number: "" })).toBeUndefined();
    expect(formatPrArtifactLabel({ number: "   " })).toBeUndefined();
    expect(formatPrArtifactLabel({ number: null })).toBeUndefined();
    expect(formatPrArtifactLabel({ prNumber: "" })).toBeUndefined();
  });
});

describe("extractArtifactMarkdownBody", () => {
  it("returns data.body when present", () => {
    expect(extractArtifactMarkdownBody({ body: "note content" })).toBe("note content");
  });

  it("returns undefined for missing / blank / non-string bodies", () => {
    expect(extractArtifactMarkdownBody(undefined)).toBeUndefined();
    expect(extractArtifactMarkdownBody({})).toBeUndefined();
    expect(extractArtifactMarkdownBody({ body: "" })).toBeUndefined();
    expect(extractArtifactMarkdownBody({ body: 123 })).toBe("123");
  });
});

describe("extractArtifactUrl", () => {
  it("returns data.url when present", () => {
    expect(extractArtifactUrl({ url: "https://example.com/pr/1" })).toBe("https://example.com/pr/1");
  });

  it("falls back to data.html_url when data.url is missing (GitHub payload shape)", () => {
    expect(extractArtifactUrl({ html_url: "https://github.com/example/repo/pull/1" })).toBe(
      "https://github.com/example/repo/pull/1",
    );
  });

  it("prefers data.url when both fields are set", () => {
    expect(
      extractArtifactUrl({
        url: "https://superplane.example/pr/1",
        html_url: "https://github.com/example/repo/pull/1",
      }),
    ).toBe("https://superplane.example/pr/1");
  });

  it("returns undefined for missing / blank / non-string urls", () => {
    expect(extractArtifactUrl(undefined)).toBeUndefined();
    expect(extractArtifactUrl({})).toBeUndefined();
    expect(extractArtifactUrl({ url: "" })).toBeUndefined();
    expect(extractArtifactUrl({ url: "   " })).toBeUndefined();
    expect(extractArtifactUrl({ url: null })).toBeUndefined();
  });
});

describe("extractArtifactTitle", () => {
  it("returns data.title when present", () => {
    expect(extractArtifactTitle({ title: "Draft PR" })).toBe("Draft PR");
  });

  it("returns undefined for missing / blank titles", () => {
    expect(extractArtifactTitle(undefined)).toBeUndefined();
    expect(extractArtifactTitle({})).toBeUndefined();
    expect(extractArtifactTitle({ title: "" })).toBeUndefined();
    expect(extractArtifactTitle({ title: "  " })).toBeUndefined();
  });
});

describe("extractArtifactName", () => {
  it("returns data.name when present", () => {
    expect(extractArtifactName({ name: "feature/refund-retry" })).toBe("feature/refund-retry");
  });

  it("returns undefined for missing / blank names", () => {
    expect(extractArtifactName(undefined)).toBeUndefined();
    expect(extractArtifactName({})).toBeUndefined();
    expect(extractArtifactName({ name: "" })).toBeUndefined();
    expect(extractArtifactName({ name: "  " })).toBeUndefined();
  });
});

describe("extractPrArtifactState", () => {
  it("returns each known state as-is", () => {
    expect(extractPrArtifactState({ state: "open" })).toBe("open");
    expect(extractPrArtifactState({ state: "draft" })).toBe("draft");
    expect(extractPrArtifactState({ state: "closed" })).toBe("closed");
    expect(extractPrArtifactState({ state: "merged" })).toBe("merged");
  });

  it("normalizes case", () => {
    expect(extractPrArtifactState({ state: "MERGED" })).toBe("merged");
    expect(extractPrArtifactState({ state: "Draft" })).toBe("draft");
  });

  it("returns undefined for missing, blank, or unrecognized values (back-compat default)", () => {
    expect(extractPrArtifactState(undefined)).toBeUndefined();
    expect(extractPrArtifactState({})).toBeUndefined();
    expect(extractPrArtifactState({ state: "" })).toBeUndefined();
    expect(extractPrArtifactState({ state: "in_review" })).toBeUndefined();
  });

  it("treats a GitHub-style `merged: true` as merged, even when state is closed", () => {
    // GitHub's own payload for a merged PR: `{ state: "closed", merged: true }`.
    // Rendering that as closed (red) would misrepresent every merged PR the
    // flow attached via the raw github.onPullRequest payload.
    expect(extractPrArtifactState({ state: "closed", merged: true })).toBe("merged");
  });

  it("treats a GitHub-style `merged: true` as merged, even when state is still open", () => {
    // Some flows attach a PR eagerly with state:"open" and only later flip
    // merged:true — before this fix the chip stayed green forever.
    expect(extractPrArtifactState({ state: "open", merged: true })).toBe("merged");
  });

  it('accepts a stringified merged flag ("true"/"false") so templated inputs work', () => {
    expect(extractPrArtifactState({ state: "closed", merged: "true" })).toBe("merged");
    expect(extractPrArtifactState({ state: "closed", merged: "TRUE" })).toBe("merged");
    expect(extractPrArtifactState({ state: "closed", merged: "false" })).toBe("closed");
  });

  it("treats a GitHub-style `draft: true` as draft when the PR is not merged", () => {
    // GitHub draft PRs are `{ state: "open", draft: true }`. Without this,
    // the chip renders as open (green) instead of the muted draft look.
    expect(extractPrArtifactState({ state: "open", draft: true })).toBe("draft");
    expect(extractPrArtifactState({ draft: "true" })).toBe("draft");
  });

  it("keeps a merged PR merged even when draft:true is also set", () => {
    // Defensive: merged is the strongest signal; a merged PR that once was
    // a draft should not flip back to draft on redisplay.
    expect(extractPrArtifactState({ merged: true, draft: true })).toBe("merged");
  });

  it("keeps an explicit non-open state over a GitHub `draft: true` flag", () => {
    expect(extractPrArtifactState({ state: "closed", draft: true })).toBe("closed");
  });

  it("does not keep a leftover state:merged when merged is explicitly false", () => {
    // A flag-only update writes `merged: false` and leaves `state: merged`
    // in the map. The chip must not stay purple.
    expect(extractPrArtifactState({ state: "merged", merged: false })).toBeUndefined();
    expect(extractPrArtifactState({ state: "merged", merged: "false" })).toBeUndefined();
  });

  it("does not keep a leftover state:draft when draft is explicitly false", () => {
    expect(extractPrArtifactState({ state: "draft", draft: false })).toBeUndefined();
  });

  it("still treats state:closed as closed when merged is explicitly false", () => {
    expect(extractPrArtifactState({ state: "closed", merged: false })).toBe("closed");
  });
});

describe("resolveBranchArtifactUrl", () => {
  it("returns the branch's own url when it has one", () => {
    expect(
      resolveBranchArtifactUrl(
        { name: "feature/refund-retry", url: "https://github.com/example/repo/tree/feature/refund-retry" },
        [],
      ),
    ).toBe("https://github.com/example/repo/tree/feature/refund-retry");
  });

  it("falls back to data.html_url on the branch itself", () => {
    expect(
      resolveBranchArtifactUrl({
        name: "feature/refund-retry",
        html_url: "https://github.com/example/repo/tree/feature/refund-retry",
      }),
    ).toBe("https://github.com/example/repo/tree/feature/refund-retry");
  });

  it("derives a GitHub tree URL from a sibling PR's URL", () => {
    // Common ordering: the flow creates the branch (attached with just a
    // name), then opens the PR later. This is what makes the branch chip
    // clickable without every flow author remembering to set `url`.
    expect(
      resolveBranchArtifactUrl({ name: "feature/refund-retry" }, [
        { type: "TYPE_PR", data: { url: "https://github.com/example/repo/pull/42" } },
      ]),
    ).toBe("https://github.com/example/repo/tree/feature/refund-retry");
  });

  it("accepts a sibling PR that only has html_url (GitHub payload shape)", () => {
    expect(
      resolveBranchArtifactUrl({ name: "hotfix/urgent" }, [
        { type: "pr", data: { html_url: "https://github.com/example/repo/pull/7" } },
      ]),
    ).toBe("https://github.com/example/repo/tree/hotfix/urgent");
  });

  it("percent-encodes each path segment of the branch name but keeps slashes", () => {
    // Branch names can contain characters the URL parser would otherwise
    // eat (e.g. `feat/#42-fix`); each segment must be encoded, but the
    // segment separator must remain a real `/`.
    expect(
      resolveBranchArtifactUrl({ name: "feat/#42-fix" }, [
        { type: "TYPE_PR", data: { url: "https://github.com/example/repo/pull/42" } },
      ]),
    ).toBe("https://github.com/example/repo/tree/feat/%2342-fix");
  });

  it("does not derive a URL for non-GitHub-style sibling URLs", () => {
    // GitLab / Bitbucket use different path prefixes (e.g. `/-/merge_requests`).
    // Building a `tree/` URL for those would ship a broken link.
    expect(
      resolveBranchArtifactUrl({ name: "feature/refund-retry" }, [
        { type: "TYPE_PR", data: { url: "https://gitlab.example.com/example/repo/-/merge_requests/1" } },
      ]),
    ).toBeUndefined();
  });

  it("ignores sibling artifacts that are not PRs", () => {
    expect(
      resolveBranchArtifactUrl({ name: "feature/refund-retry" }, [
        { type: "TYPE_MARKDOWN", data: { url: "https://github.com/example/repo/pull/42" } },
      ]),
    ).toBeUndefined();
  });

  it("returns undefined when the branch has neither a url nor a name", () => {
    expect(resolveBranchArtifactUrl({})).toBeUndefined();
    expect(resolveBranchArtifactUrl(undefined)).toBeUndefined();
  });

  it("returns undefined when there is no sibling PR at all", () => {
    expect(resolveBranchArtifactUrl({ name: "feature/refund-retry" }, [])).toBeUndefined();
    expect(resolveBranchArtifactUrl({ name: "feature/refund-retry" }, undefined)).toBeUndefined();
  });

  it("does not guess when multiple sibling PRs point at different repos", () => {
    // Two GitHub PRs and no head-ref match: picking the first would
    // send the branch chip to the wrong repository.
    expect(
      resolveBranchArtifactUrl({ name: "feature/refund-retry" }, [
        { type: "TYPE_PR", data: { url: "https://github.com/acme/payments/pull/1" } },
        { type: "TYPE_PR", data: { url: "https://github.com/acme/storefront/pull/9" } },
      ]),
    ).toBeUndefined();
  });

  it("prefers the sibling PR whose head ref matches the branch name", () => {
    expect(
      resolveBranchArtifactUrl({ name: "feature/refund-retry" }, [
        { type: "TYPE_PR", data: { url: "https://github.com/acme/payments/pull/1", head_ref: "chore/deps" } },
        {
          type: "TYPE_PR",
          data: { url: "https://github.com/acme/storefront/pull/9", head: { ref: "feature/refund-retry" } },
        },
      ]),
    ).toBe("https://github.com/acme/storefront/tree/feature/refund-retry");
  });
});

describe("normalizeArtifactKind", () => {
  it("strips the TYPE_ proto prefix and lower-cases the value", () => {
    expect(normalizeArtifactKind("TYPE_PR")).toBe("pr");
    expect(normalizeArtifactKind("TYPE_BRANCH")).toBe("branch");
    expect(normalizeArtifactKind("branch")).toBe("branch");
    expect(normalizeArtifactKind("")).toBe("");
  });
});

describe("buildLatestArtifactDataById", () => {
  it("indexes artifacts that have both an id and data", () => {
    const byId = buildLatestArtifactDataById([
      { id: "art-1", data: { state: "merged" } },
      { id: "art-2" },
      { data: { state: "open" } },
    ]);

    expect(byId.get("art-1")).toEqual({ state: "merged" });
    expect(byId.has("art-2")).toBe(false);
  });
});

describe("overlayLiveArtifactData", () => {
  it("replaces snapshot data when a matching live row exists", () => {
    const snapshot = { id: "art-1", type: "pr", data: { state: "open" } };
    const live = overlayLiveArtifactData(snapshot, new Map([["art-1", { state: "merged", title: "Done" }]]));

    expect(live).toEqual({ id: "art-1", type: "pr", data: { state: "merged", title: "Done" } });
  });

  it("keeps the snapshot when the artifact is missing from the live list", () => {
    const snapshot = { id: "art-1", data: { state: "open" } };
    expect(overlayLiveArtifactData(snapshot, new Map())).toBe(snapshot);
  });
});

describe("toArtifactDataRecord", () => {
  it("returns a record for object payloads and undefined otherwise", () => {
    expect(toArtifactDataRecord({ url: "https://example.com" })).toEqual({ url: "https://example.com" });
    expect(toArtifactDataRecord(undefined)).toBeUndefined();
    expect(toArtifactDataRecord("https://example.com")).toBeUndefined();
  });
});

describe("toWorkOrderArtifactLikes", () => {
  it("narrows API artifacts to the sibling-lookup shape", () => {
    expect(
      toWorkOrderArtifactLikes([
        { type: "TYPE_PR", data: { url: "https://github.com/example/repo/pull/1" } },
        { type: null, data: undefined },
      ]),
    ).toEqual([
      { type: "TYPE_PR", data: { url: "https://github.com/example/repo/pull/1" } },
      { type: undefined, data: undefined },
    ]);
  });
});
