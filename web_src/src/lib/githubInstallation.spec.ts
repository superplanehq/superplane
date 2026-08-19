import { describe, expect, it } from "vitest";

import { githubInstallationUrl } from "./githubInstallation";

describe("githubInstallationUrl", () => {
  it("uses the URL the GitHub App setup stored", () => {
    expect(
      githubInstallationUrl({
        status: {
          properties: [
            { name: "appInstallationURL", value: "https://github.com/organizations/acme/settings/installations/42" },
          ],
        },
      }),
    ).toBe("https://github.com/organizations/acme/settings/installations/42");
  });

  it("rebuilds the organization page for an older integration", () => {
    expect(
      githubInstallationUrl({
        spec: { configuration: { organization: "acme" } },
        status: { metadata: { installationId: "42" } },
      }),
    ).toBe("https://github.com/organizations/acme/settings/installations/42");
  });

  it("rebuilds the personal account page when no organization is set", () => {
    expect(
      githubInstallationUrl({
        spec: { configuration: { organization: "" } },
        status: { metadata: { installationId: "42" } },
      }),
    ).toBe("https://github.com/settings/installations/42");
  });

  it("falls back to the list of installations when the installation is unknown", () => {
    expect(githubInstallationUrl(null)).toBe("https://github.com/settings/installations");
    expect(githubInstallationUrl({ status: { metadata: {} } })).toBe("https://github.com/settings/installations");
  });
});
