import { describe, expect, it } from "bun:test";

import { noteActionDisabled } from "./splitRunNoteActionStyle";

describe("noteActionDisabled", () => {
  it("disables Rerun when the action sets disabled", () => {
    expect(
      noteActionDisabled("rerun", {
        actionBusy: false,
        startBusy: false,
        startDisabled: false,
        actionDisabled: true,
      }),
    ).toBe(true);
  });

  it("keeps Rerun enabled when only Start is locked", () => {
    expect(
      noteActionDisabled("rerun", {
        actionBusy: false,
        startBusy: true,
        startDisabled: true,
        actionDisabled: false,
      }),
    ).toBe(false);
  });

  it("disables Start for startDisabled, startBusy, or actionDisabled", () => {
    expect(
      noteActionDisabled("start", {
        actionBusy: false,
        startBusy: false,
        startDisabled: true,
        actionDisabled: false,
      }),
    ).toBe(true);
    expect(
      noteActionDisabled("start", {
        actionBusy: false,
        startBusy: true,
        startDisabled: false,
        actionDisabled: false,
      }),
    ).toBe(true);
    expect(
      noteActionDisabled("start", {
        actionBusy: false,
        startBusy: false,
        startDisabled: false,
        actionDisabled: true,
      }),
    ).toBe(true);
  });
});
