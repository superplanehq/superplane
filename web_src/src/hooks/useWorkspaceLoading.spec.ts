import { describe, expect, it } from "bun:test";

import { lastLoadingMessage, nextLoadingMessages } from "./useWorkspaceLoading";

describe("nextLoadingMessages", () => {
  it("keeps insertion order so the latest pending message wins", () => {
    const afterAccount = nextLoadingMessages({}, "account", "Loading account", true);
    const afterBoard = nextLoadingMessages(afterAccount, "board", "Loading the board", true);

    expect(lastLoadingMessage(afterBoard)).toBe("Loading the board");

    const afterAccountDone = nextLoadingMessages(afterBoard, "account", "Loading account", false);
    expect(lastLoadingMessage(afterAccountDone)).toBe("Loading the board");
    expect(afterAccountDone).not.toHaveProperty("account");
  });

  it("returns the same object when the entry does not change", () => {
    const current = { account: "Loading account" };
    expect(nextLoadingMessages(current, "account", "Loading account", true)).toBe(current);
    expect(nextLoadingMessages(current, "missing", "Loading the board", false)).toBe(current);
  });
});
