import { describe, expect, it } from "vitest";
import { validateReplayLaunch, type ReplayInputDraft } from "./validateReplayLaunch";

function draft(overrides: Partial<ReplayInputDraft> = {}): ReplayInputDraft {
  return {
    sourceNodeId: "node-a",
    label: "node-a",
    editedText: "{}",
    status: undefined,
    ...overrides,
  };
}

describe("validateReplayLaunch", () => {
  it("builds N inputs, with the edited one differing and the rest matching history", () => {
    const drafts = [
      draft({ sourceNodeId: "node-a", label: "node-a", editedText: '{"x":1}', status: "STATUS_RECOVERED" }),
      draft({ sourceNodeId: "node-b", label: "node-b", editedText: '{"y":9}', status: "STATUS_RECOVERED" }),
    ];

    const result = validateReplayLaunch(drafts);

    expect(result.valid).toBe(true);
    expect(result.request?.inputs).toEqual([
      { payload: { x: 1 }, sourceNodeId: "node-a" },
      { payload: { y: 9 }, sourceNodeId: "node-b" },
    ]);
  });

  it("sends a lone input as an attributed input, so an edited payload is not overridden by history", () => {
    const drafts = [
      draft({ sourceNodeId: "node-a", label: "node-a", editedText: '{"edited":true}', status: "STATUS_RECOVERED" }),
    ];

    const result = validateReplayLaunch(drafts);

    expect(result.valid).toBe(true);
    expect(result.request?.inputs).toEqual([{ payload: { edited: true }, sourceNodeId: "node-a" }]);
  });

  it("blocks launch when every input is detached, instead of sending nothing replayable", () => {
    const drafts = [
      draft({ sourceNodeId: "node-a", label: "node-a", editedText: '{"stale":1}', status: "STATUS_DETACHED" }),
      draft({ sourceNodeId: "node-b", label: "node-b", editedText: '{"stale":2}', status: "STATUS_DETACHED" }),
    ];

    const result = validateReplayLaunch(drafts);

    expect(result.valid).toBe(false);
    expect(result.request).toBeNull();
    expect(result.launchError).toContain("no longer feed");
  });

  it("never includes a detached input in the request", () => {
    const drafts = [
      draft({ sourceNodeId: "node-a", label: "node-a", editedText: '{"x":1}', status: "STATUS_RECOVERED" }),
      draft({ sourceNodeId: "node-b", label: "node-b", editedText: '{"y":9}', status: "STATUS_RECOVERED" }),
      draft({ sourceNodeId: "node-c", label: "node-c", editedText: '{"stale":true}', status: "STATUS_DETACHED" }),
    ];

    const result = validateReplayLaunch(drafts);

    expect(result.valid).toBe(true);
    expect(result.request?.inputs).toHaveLength(2);
    expect(result.request?.inputs?.some((input) => input.sourceNodeId === "node-c")).toBe(false);
  });

  it("never blocks launch on invalid JSON left in a detached input", () => {
    const drafts = [
      draft({ sourceNodeId: "node-a", label: "node-a", editedText: '{"x":1}', status: "STATUS_RECOVERED" }),
      draft({ sourceNodeId: "node-b", label: "node-b", editedText: '{"y":9}', status: "STATUS_RECOVERED" }),
      draft({ sourceNodeId: "node-c", label: "node-c", editedText: "not json at all", status: "STATUS_DETACHED" }),
    ];

    const result = validateReplayLaunch(drafts);

    expect(result.valid).toBe(true);
  });

  it("no longer blocks once a missing input has been filled in, and sends its payload", () => {
    const drafts = [
      draft({ sourceNodeId: "node-a", label: "node-a", editedText: '{"x":1}', status: "STATUS_RECOVERED" }),
      draft({ sourceNodeId: "node-b", label: "node-b", editedText: '{"filled":true}', status: "STATUS_MISSING" }),
    ];

    const result = validateReplayLaunch(drafts);

    expect(result.valid).toBe(true);
    expect(result.request?.inputs).toEqual([
      { payload: { x: 1 }, sourceNodeId: "node-a" },
      { payload: { filled: true }, sourceNodeId: "node-b" },
    ]);
  });

  it("blocks launch and names the unfilled source when a missing input is left empty", () => {
    const drafts = [
      draft({ sourceNodeId: "node-a", label: "node-a", editedText: '{"x":1}', status: "STATUS_RECOVERED" }),
      draft({ sourceNodeId: "node-b", label: "node-b", editedText: "{}", status: "STATUS_MISSING" }),
    ];

    const result = validateReplayLaunch(drafts);

    expect(result.valid).toBe(false);
    expect(result.request).toBeNull();
    expect(result.fields[1]).toEqual({ valid: false, error: expect.stringContaining("node-b"), kind: "missing" });
  });

  it("blocks launch and points to the specific editor with invalid JSON", () => {
    const drafts = [
      draft({ sourceNodeId: "node-a", label: "node-a", editedText: "{ not valid json", status: "STATUS_RECOVERED" }),
      draft({ sourceNodeId: "node-b", label: "node-b", editedText: '{"y":9}', status: "STATUS_RECOVERED" }),
    ];

    const result = validateReplayLaunch(drafts);

    expect(result.valid).toBe(false);
    expect(result.fields[0].valid).toBe(false);
    expect(result.fields[1].valid).toBe(true);
    expect((result.fields[0] as { valid: false; error: string }).error).toContain("node-a");
  });
});
