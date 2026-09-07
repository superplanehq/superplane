import { describe, expect, it } from "vitest";

import { organizationNameFromAccount } from "./organizationNameFromAccount";

describe("organizationNameFromAccount", () => {
  it("uses the SuperPlane account name before GitHub connect", () => {
    expect(
      organizationNameFromAccount({
        id: "account-1",
        name: "Dev User",
        email: "dev@superplane.local",
        avatar_url: "",
        installation_admin: false,
        has_password: true,
        linked_accounts: [{ provider: "github", name: "GitHub Owner", username: "dev-user" }],
        providers: [{ provider: "github", username: "dev-user" }],
      }),
    ).toBe("Dev User");
  });

  it("uses the email local part when the account name is empty", () => {
    expect(
      organizationNameFromAccount({
        id: "account-1",
        name: "   ",
        email: "dev@superplane.local",
        avatar_url: "",
        installation_admin: false,
        has_password: true,
        linked_accounts: [],
        providers: [],
      }),
    ).toBe("dev");
  });
});
