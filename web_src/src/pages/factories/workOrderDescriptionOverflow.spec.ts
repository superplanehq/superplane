import { describe, expect, it } from "vitest";

import {
  collapsedDescriptionMaxHeight,
  descriptionLeftoverCapacity,
  descriptionNeedsCollapse,
  descriptionPaneCapacity,
  nearestScrollParent,
  reservedPaneSiblingHeight,
} from "./workOrderDescriptionOverflow";

describe("descriptionPaneCapacity", () => {
  it("falls back to 220px when there is no scroll pane", () => {
    expect(descriptionPaneCapacity(null)).toBe(220);
  });

  it("falls back to 220px when the pane has not been measured yet", () => {
    expect(descriptionPaneCapacity({ clientHeight: 0, paddingTop: 24, paddingBottom: 24 })).toBe(220);
  });

  it("uses the inner height of the scroll pane", () => {
    expect(descriptionPaneCapacity({ clientHeight: 520, paddingTop: 24, paddingBottom: 24 })).toBe(472);
  });
});

describe("descriptionNeedsCollapse", () => {
  it("keeps the body open when it fits the pane", () => {
    expect(descriptionNeedsCollapse(400, 472)).toBe(false);
  });

  it("collapses only when the body is taller than the pane", () => {
    expect(descriptionNeedsCollapse(600, 472)).toBe(true);
  });
});

describe("descriptionLeftoverCapacity", () => {
  it("subtracts reserved siblings such as checks", () => {
    expect(descriptionLeftoverCapacity(472, 160)).toBe(312);
  });

  it("falls back to 220px when siblings already fill the pane", () => {
    expect(descriptionLeftoverCapacity(472, 500)).toBe(220);
  });
});

describe("reservedPaneSiblingHeight", () => {
  it("includes later siblings and their margins", () => {
    const pane = document.createElement("div");
    pane.style.overflowY = "auto";
    const block = document.createElement("div");
    const content = document.createElement("div");
    block.append(content);
    const checks = document.createElement("div");
    checks.style.marginTop = "40px";
    Object.defineProperty(checks, "offsetHeight", { configurable: true, get: () => 120 });
    pane.append(block, checks);
    document.body.append(pane);

    expect(reservedPaneSiblingHeight(content, pane)).toBe(160);

    pane.remove();
  });
});

describe("collapsedDescriptionMaxHeight", () => {
  it("leaves room for the Show more control", () => {
    expect(collapsedDescriptionMaxHeight(472)).toBe(440);
  });

  it("keeps at least 100px of description when collapsed", () => {
    expect(collapsedDescriptionMaxHeight(50)).toBe(100);
  });
});

describe("nearestScrollParent", () => {
  it("finds the nearest overflow auto ancestor", () => {
    const pane = document.createElement("div");
    pane.style.overflowY = "auto";
    const content = document.createElement("div");
    pane.append(content);
    document.body.append(pane);

    expect(nearestScrollParent(content)).toBe(pane);

    pane.remove();
  });
});
