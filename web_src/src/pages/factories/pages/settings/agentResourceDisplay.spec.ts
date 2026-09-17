import { describe, expect, it } from "bun:test";

import {
  HEADER_MCP_RESOURCE,
  OAUTH_CONNECTED_RESOURCE,
  OAUTH_NEEDS_RECONNECT_RESOURCE,
  OAUTH_NOT_CONNECTED_RESOURCE,
  OAUTH_VENDOR_REJECTED_RESOURCE,
  UI_UX_PRO_MAX_SKILL,
} from "../../__fixtures__/agentResourceFixtures";
import {
  connectionAuthLabel,
  connectionNeedsOAuthAction,
  connectionStatusLabel,
  skillSourceLabel,
} from "./agentResourceDisplay";

describe("connectionAuthLabel", () => {
  it("labels header and sign-in auth", () => {
    expect(connectionAuthLabel("AUTH_HEADERS")).toBe("Header");
    expect(connectionAuthLabel("AUTH_OAUTH")).toBe("Sign-in");
  });
});

describe("connectionStatusLabel", () => {
  it("treats header auth as connected", () => {
    expect(connectionStatusLabel(HEADER_MCP_RESOURCE)).toBe("Connected");
  });

  it("maps oauth statuses", () => {
    expect(connectionStatusLabel(OAUTH_NOT_CONNECTED_RESOURCE)).toBe("Not connected");
    expect(connectionStatusLabel(OAUTH_CONNECTED_RESOURCE)).toBe("Connected");
    expect(connectionStatusLabel(OAUTH_NEEDS_RECONNECT_RESOURCE)).toBe("Reconnect");
    expect(connectionStatusLabel(OAUTH_VENDOR_REJECTED_RESOURCE)).toBe("Reconnect");
  });
});

describe("connectionNeedsOAuthAction", () => {
  it("is true when sign-in is not connected", () => {
    expect(connectionNeedsOAuthAction(HEADER_MCP_RESOURCE)).toBe(false);
    expect(connectionNeedsOAuthAction(OAUTH_CONNECTED_RESOURCE)).toBe(false);
    expect(connectionNeedsOAuthAction(OAUTH_NOT_CONNECTED_RESOURCE)).toBe(true);
    expect(connectionNeedsOAuthAction(OAUTH_VENDOR_REJECTED_RESOURCE)).toBe(true);
  });
});

describe("skillSourceLabel", () => {
  it("formats a GitHub package as owner/repo@ref", () => {
    expect(skillSourceLabel(UI_UX_PRO_MAX_SKILL)).toBe("nextlevelbuilder/ui-ux-pro-max-skill@v1.2.0");
  });
});
