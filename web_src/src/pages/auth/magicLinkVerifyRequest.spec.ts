import { describe, expect, it } from "vitest";
import { buildMagicLinkVerifyRequest } from "./magicLinkVerifyRequest";

describe("buildMagicLinkVerifyRequest", () => {
  it("includes signup intent for signup magic links", () => {
    const request = buildMagicLinkVerifyRequest({
      token: "token-123",
      inviteToken: "",
      redirectTarget: "/invite/abc",
      signupIntent: true,
    });

    expect(request.url).toBe("/auth/magic-code/verify?redirect=%2Finvite%2Fabc");
    expect(new URLSearchParams(request.body).get("token")).toBe("token-123");
    expect(new URLSearchParams(request.body).get("signup")).toBe("true");
  });

  it("does not include signup intent for login magic links", () => {
    const request = buildMagicLinkVerifyRequest({
      token: "token-123",
      inviteToken: "",
      redirectTarget: "",
      signupIntent: false,
    });

    expect(request.url).toBe("/auth/magic-code/verify");
    expect(new URLSearchParams(request.body).get("signup")).toBeNull();
  });

  it("includes invite token when present", () => {
    const request = buildMagicLinkVerifyRequest({
      token: "token-123",
      inviteToken: "invite-123",
      redirectTarget: "/invite/invite-123",
      signupIntent: false,
    });

    expect(new URLSearchParams(request.body).get("invite_token")).toBe("invite-123");
  });
});
