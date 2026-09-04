import { describe, expect, it } from "vitest";

import {
  GITHUB_INSTALL_REQUEST_NEXT,
  githubInstallRequestBody,
  githubInstallRequestSettingsTitle,
} from "./githubInstallRequestCopy";

describe("githubInstallRequestCopy", () => {
  it("names the GitHub organization when SuperPlane knows it", () => {
    expect(githubInstallRequestBody("acme")).toBe("Ask an admin of acme to approve the SuperPlane GitHub App.");
    expect(githubInstallRequestSettingsTitle("acme")).toBe("Waiting for acme approval");
  });

  it("falls back when the organization is unknown", () => {
    expect(githubInstallRequestBody()).toBe("Ask a GitHub organization admin to approve the SuperPlane GitHub App.");
    expect(githubInstallRequestSettingsTitle()).toBe("Waiting for GitHub approval");
    expect(GITHUB_INSTALL_REQUEST_NEXT).toBe("After they approve, click Connect GitHub again.");
  });
});
