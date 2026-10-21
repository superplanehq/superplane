import { describe, expect, it } from "bun:test";

import { isGitHubUserAttachmentUrl } from "./githubAttachments";

describe("isGitHubUserAttachmentUrl", () => {
  it("accepts github.com user attachments", () => {
    expect(isGitHubUserAttachmentUrl("https://github.com/user-attachments/assets/abc")).toBe(true);
  });

  it("accepts GitHub Enterprise Cloud attachments", () => {
    expect(isGitHubUserAttachmentUrl("https://acme.ghe.com/user-attachments/assets/abc")).toBe(true);
  });

  it("accepts a matching GitHub Enterprise Server host", () => {
    expect(
      isGitHubUserAttachmentUrl("https://ghes.corp.example/user-attachments/assets/abc", "ghes.corp.example"),
    ).toBe(true);
  });

  it("rejects a foreign host", () => {
    expect(isGitHubUserAttachmentUrl("https://example.com/user-attachments/assets/abc")).toBe(false);
  });
});
