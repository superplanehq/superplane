import { describe, expect, it } from "vitest";

import { apiPanelTypeToPanelType, panelTypeToApi } from "./apiPanelType";
import { PANEL_TYPES } from "./panelTypes";

describe("apiPanelTypeToPanelType", () => {
  it("maps the SCREAMING_CASE SDK enum to the lowercase FE form", () => {
    expect(apiPanelTypeToPanelType("MARKDOWN")).toBe("markdown");
    expect(apiPanelTypeToPanelType("NODE")).toBe("node");
    expect(apiPanelTypeToPanelType("NODES")).toBe("nodes");
    expect(apiPanelTypeToPanelType("TABLE")).toBe("table");
    expect(apiPanelTypeToPanelType("CHART")).toBe("chart");
    expect(apiPanelTypeToPanelType("NUMBER")).toBe("number");
  });

  it("passes through the lowercase form the previous API sent as plain strings", () => {
    for (const type of PANEL_TYPES) {
      expect(apiPanelTypeToPanelType(type)).toBe(type);
    }
  });

  it("returns undefined for unset / unspecified / unknown values", () => {
    expect(apiPanelTypeToPanelType(undefined)).toBeUndefined();
    expect(apiPanelTypeToPanelType(null)).toBeUndefined();
    expect(apiPanelTypeToPanelType("")).toBeUndefined();
    expect(apiPanelTypeToPanelType("TYPE_UNSPECIFIED")).toBeUndefined();
    expect(apiPanelTypeToPanelType("bogus")).toBeUndefined();
  });
});

describe("apiPanelTypeToPanelType / panelTypeToApi round-trip", () => {
  it("round-trips every panel type through the SDK enum", () => {
    for (const type of PANEL_TYPES) {
      expect(apiPanelTypeToPanelType(panelTypeToApi(type))).toBe(type);
    }
  });
});
