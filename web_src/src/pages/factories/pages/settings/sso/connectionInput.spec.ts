import { describe, expect, it } from "bun:test";

import { ssoConnectionInput, type SsoForm } from "./useSsoSettingsState";

function form(overrides: Partial<SsoForm> = {}): SsoForm {
  return {
    domain: "",
    displayName: "Okta",
    preset: "PRESET_OKTA",
    issuer: "https://example.okta.com",
    clientId: "client",
    clientSecret: "",
    jitEnabled: true,
    defaultRole: "org_operator",
    enforcement: "ENFORCEMENT_OPTIONAL",
    ownerBypass: true,
    sessionHours: "24",
    idpLogout: false,
    tokenDescription: "Okta",
    scopes: ["openid", "groups"],
    subjectClaim: "uid",
    emailClaim: "mail",
    nameClaim: "displayName",
    loadedPreset: "PRESET_OKTA",
    ...overrides,
  };
}

describe("ssoConnectionInput", () => {
  it("sends the loaded claims on each save", () => {
    const input = ssoConnectionInput(form(), "ENFORCEMENT_OPTIONAL", 86400);
    expect(input.scopes).toEqual(["openid", "groups"]);
    expect(input.subjectClaim).toBe("uid");
    expect(input.emailClaim).toBe("mail");
    expect(input.nameClaim).toBe("displayName");
  });

  it("clears the claims when the preset changes", () => {
    const input = ssoConnectionInput(form({ preset: "PRESET_GENERIC" }), "ENFORCEMENT_OPTIONAL", 86400);
    expect(input.scopes).toEqual([]);
    expect(input.subjectClaim).toBe("");
    expect(input.emailClaim).toBe("");
    expect(input.nameClaim).toBe("");
  });
});
