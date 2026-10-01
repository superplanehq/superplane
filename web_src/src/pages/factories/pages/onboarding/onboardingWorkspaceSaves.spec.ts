import { describe, expect, it, vi } from "bun:test";

import { WORKSPACE_STILL_OPENING, guardWorkspaceSave } from "./onboardingWorkspaceSaves";

describe("guardWorkspaceSave", () => {
  it("does not call the previous workspace save while the new lookup is closed", async () => {
    const save = vi.fn().mockResolvedValue("saved");
    const guarded = guardWorkspaceSave(() => false, save);

    await expect(guarded("name")).rejects.toThrow(WORKSPACE_STILL_OPENING);
    expect(save).not.toHaveBeenCalled();
  });

  it("calls the save when the open workspace matches the lookup", async () => {
    const save = vi.fn().mockResolvedValue("saved");
    const guarded = guardWorkspaceSave(() => true, save);

    await expect(guarded("name")).resolves.toBe("saved");
    expect(save).toHaveBeenCalledWith("name");
  });
});
