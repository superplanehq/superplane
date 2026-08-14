import { describe, expect, it } from "vitest";

import { factoryLineDestinationPath } from "./factoryLineNavigation";

const destination = {
  organizationId: "organization",
  factoryKey: "factory",
  lineId: "line",
};

describe("factoryLineDestinationPath", () => {
  it("sends editors to the line editor", () => {
    expect(factoryLineDestinationPath({ ...destination, canEdit: true })).toBe(
      "/organization/workspaces/factory/lines/line/edit",
    );
  });

  it("sends viewers to the readable line page", () => {
    expect(factoryLineDestinationPath({ ...destination, canEdit: false })).toBe(
      "/organization/workspaces/factory/lines/line",
    );
  });
});
