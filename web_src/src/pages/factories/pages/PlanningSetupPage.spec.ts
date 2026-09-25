import { describe, expect, it } from "bun:test";

import { planningSetupRedirect } from "./planningSetupCaption";

describe("planningSetupRedirect", () => {
  it("returns to the board when the user cannot configure Planning", () => {
    expect(
      planningSetupRedirect({
        canUpdate: false,
        lineId: "line-1",
        factoryPresent: true,
        linePresent: true,
        boardHref: "/org-1/workspaces/rf/lines/line-1",
      }),
    ).toBe("/org-1/workspaces/rf/lines/line-1");
  });

  it("returns to the board when the line is missing", () => {
    expect(
      planningSetupRedirect({
        canUpdate: true,
        lineId: "missing",
        factoryPresent: true,
        linePresent: false,
        boardHref: "/org-1/workspaces/rf/lines/line-1",
      }),
    ).toBe("/org-1/workspaces/rf/lines/line-1");
  });

  it("stays on the wizard when the line is present", () => {
    expect(
      planningSetupRedirect({
        canUpdate: true,
        lineId: "line-1",
        factoryPresent: true,
        linePresent: true,
        boardHref: "/org-1/workspaces/rf/lines/line-1",
      }),
    ).toBeUndefined();
  });
});
