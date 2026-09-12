import { describe, expect, it } from "bun:test";

import {
  GITHUB_INSTALL_APPROVED_ACTION,
  GITHUB_INSTALL_APPROVED_BODY,
  GITHUB_INSTALL_APPROVED_TITLE,
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
    expect(GITHUB_INSTALL_REQUEST_NEXT).toBe("SuperPlane checks for approval automatically.");
  });

  it("tells a GitHub admin the request is approved", () => {
    expect(GITHUB_INSTALL_APPROVED_TITLE).toBe("Request approved");
    expect(GITHUB_INSTALL_APPROVED_BODY).toBe(
      "The SuperPlane GitHub App is approved. The person who asked can continue in SuperPlane.",
    );
    expect(GITHUB_INSTALL_APPROVED_ACTION).toBe("Open SuperPlane");
  });
});
