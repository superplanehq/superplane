import { beforeEach, describe, expect, it } from "bun:test";

import {
  LAST_USED_LOGIN_METHOD_STORAGE_KEY,
  readLastUsedLoginMethod,
  recordLastUsedLoginMethod,
} from "./lastUsedLoginMethod";

describe("lastUsedLoginMethod", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("stores SSO as a known method", () => {
    recordLastUsedLoginMethod("sso");
    expect(window.localStorage.getItem(LAST_USED_LOGIN_METHOD_STORAGE_KEY)).toBe("sso");
    expect(readLastUsedLoginMethod()).toBe("sso");
  });

  it("ignores an unknown stored method", () => {
    window.localStorage.setItem(LAST_USED_LOGIN_METHOD_STORAGE_KEY, "saml");
    expect(readLastUsedLoginMethod()).toBeNull();
  });
});
