import { describe, expect, it } from "bun:test";

import { githubRepositorySecurityAnalysisUrl } from "./githubRepository";

describe("githubRepositorySecurityAnalysisUrl", () => {
  it("builds the security analysis settings URL", () => {
    expect(githubRepositorySecurityAnalysisUrl("operately/website")).toBe(
      "https://github.com/operately/website/settings/security_analysis",
    );
  });

  it("returns undefined for non-owner/repo values", () => {
    expect(githubRepositorySecurityAnalysisUrl("")).toBeUndefined();
    expect(githubRepositorySecurityAnalysisUrl("Workspace backlog repository")).toBeUndefined();
  });
});
