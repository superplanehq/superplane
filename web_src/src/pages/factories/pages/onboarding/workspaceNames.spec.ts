import { describe, expect, it } from "vitest";

import { isPlaceholderWorkspaceName, placeholderWorkspaceName, workspaceNameFromRepository } from "./workspaceNames";

describe("placeholderWorkspaceName", () => {
  it("uses the base name when the organization has no workspace with it", () => {
    expect(placeholderWorkspaceName(["Payments"])).toBe("New workspace");
  });

  it("counts up until the name is free", () => {
    expect(placeholderWorkspaceName(["New workspace", "new workspace 2"])).toBe("New workspace 3");
  });
});

describe("isPlaceholderWorkspaceName", () => {
  it("recognizes the base placeholder and numbered variants", () => {
    expect(isPlaceholderWorkspaceName("New workspace")).toBe(true);
    expect(isPlaceholderWorkspaceName("new workspace 3")).toBe(true);
    expect(isPlaceholderWorkspaceName("Payments Service")).toBe(false);
  });
});

describe("workspaceNameFromRepository", () => {
  it("turns the repository name into a readable workspace name", () => {
    expect(workspaceNameFromRepository("acme/payments-service")).toBe("Payments Service");
    expect(workspaceNameFromRepository("acme/mobile_ios")).toBe("Mobile Ios");
  });

  it("returns an empty name when the repository has none", () => {
    expect(workspaceNameFromRepository("")).toBe("");
  });
});
