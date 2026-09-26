import { describe, expect, it } from "bun:test";

import { bearerAuthorizationValue, catalogHeaderSecretName } from "./mcpHeaderAuth";

describe("bearerAuthorizationValue", () => {
  it("adds Bearer when the token has no prefix", () => {
    expect(bearerAuthorizationValue("  ghp_example  ")).toBe("Bearer ghp_example");
  });

  it("does not add Bearer twice", () => {
    expect(bearerAuthorizationValue("Bearer ghp_example")).toBe("Bearer ghp_example");
    expect(bearerAuthorizationValue("bearer ghp_example")).toBe("Bearer ghp_example");
  });
});

describe("catalogHeaderSecretName", () => {
  it("names the secret from the server name and a unique part", () => {
    expect(catalogHeaderSecretName("github", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")).toBe(
      "github-mcp-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    );
  });
});
