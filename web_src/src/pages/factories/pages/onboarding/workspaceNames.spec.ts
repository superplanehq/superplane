import { describe, expect, it } from "vitest";

import { isPlaceholderWorkspaceName, uniqueWorkspaceName, workspaceNameFromRepository } from "./workspaceNames";

describe("uniqueWorkspaceName", () => {
  it("keeps the name when the organization has no workspace with it", () => {
    expect(uniqueWorkspaceName("New workspace", ["Payments"])).toBe("New workspace");
  });

  it("counts up until the name is free", () => {
    expect(uniqueWorkspaceName("New workspace", ["New workspace", "new workspace 2"])).toBe("New workspace 3");
  });

  it("ignores case and surrounding spaces", () => {
    expect(uniqueWorkspaceName("  Payments Service ", ["payments service"])).toBe("Payments Service 2");
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
