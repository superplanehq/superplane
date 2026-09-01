import { describe, expect, it } from "vitest";

import { accountEmailOptions, accountEmailSourceLabel, ssoLinkHref } from "./accountSettings";

describe("accountEmailOptions", () => {
  it("lists unique emails from connected sign-in methods", () => {
    expect(
      accountEmailOptions({
        email: "ada@example.com",
        hasPassword: true,
        providers: [
          { provider: "github", email: "ada@users.noreply.github.com" },
          { provider: "google", email: "ada@example.com" },
        ],
      }),
    ).toEqual([
      { email: "ada@example.com", sources: ["password", "google"] },
      { email: "ada@users.noreply.github.com", sources: ["github"] },
    ]);
    expect(accountEmailSourceLabel(["github", "google"])).toBe("GitHub · Google");
  });
});

describe("ssoLinkHref", () => {
  it("sends the user to an authenticated link start", () => {
    expect(ssoLinkHref("google", "/org/workspaces/RF/settings/account/security")).toBe(
      "/auth/google?intent=link&redirect=%2Forg%2Fworkspaces%2FRF%2Fsettings%2Faccount%2Fsecurity",
    );
  });
});
