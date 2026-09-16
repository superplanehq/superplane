import { describe, expect, it, beforeEach } from "bun:test";

import {
  LAST_VISITED_LOCATION_STORAGE_KEY,
  readLastVisitedLocation,
  recordLastVisitedLocation,
} from "./lastVisitedLocation";

describe("lastVisitedLocation", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("returns null when nothing was recorded", () => {
    expect(readLastVisitedLocation("account-1", "acme")).toBeNull();
  });

  it("records and reads the last visited path per account and organization", () => {
    recordLastVisitedLocation("account-1", "acme", "/acme/apps/deploy?run=1");
    recordLastVisitedLocation("account-1", "widgets", "/widgets/apps/build?run=2");

    expect(readLastVisitedLocation("account-1", "acme")).toBe("/acme/apps/deploy?run=1");
    expect(readLastVisitedLocation("account-1", "widgets")).toBe("/widgets/apps/build?run=2");
  });

  it("overwrites the previous path for the same account and organization", () => {
    recordLastVisitedLocation("account-1", "acme", "/acme/apps/deploy?run=1");
    recordLastVisitedLocation("account-1", "acme", "/acme/apps/deploy?run=2&node=approve-1");

    expect(readLastVisitedLocation("account-1", "acme")).toBe("/acme/apps/deploy?run=2&node=approve-1");
  });

  it("does not mix up different accounts in the same organization", () => {
    recordLastVisitedLocation("account-1", "acme", "/acme/apps/deploy?run=1");
    recordLastVisitedLocation("account-2", "acme", "/acme/apps/build?run=2");

    expect(readLastVisitedLocation("account-1", "acme")).toBe("/acme/apps/deploy?run=1");
    expect(readLastVisitedLocation("account-2", "acme")).toBe("/acme/apps/build?run=2");
  });

  it("ignores unsafe paths", () => {
    recordLastVisitedLocation("account-1", "acme", "//evil.com");
    expect(readLastVisitedLocation("account-1", "acme")).toBeNull();
  });

  it("ignores malformed stored values", () => {
    window.localStorage.setItem(LAST_VISITED_LOCATION_STORAGE_KEY, "not-json");
    expect(readLastVisitedLocation("account-1", "acme")).toBeNull();

    window.localStorage.setItem(LAST_VISITED_LOCATION_STORAGE_KEY, JSON.stringify(["/acme"]));
    expect(readLastVisitedLocation("account-1", "acme")).toBeNull();

    window.localStorage.setItem(LAST_VISITED_LOCATION_STORAGE_KEY, JSON.stringify({ "account-1:acme": "//evil.com" }));
    expect(readLastVisitedLocation("account-1", "acme")).toBeNull();
  });

  it("ignores empty account ids or organization routes", () => {
    recordLastVisitedLocation("", "acme", "/acme");
    recordLastVisitedLocation("account-1", "", "/acme");

    expect(readLastVisitedLocation("", "acme")).toBeNull();
    expect(readLastVisitedLocation("account-1", "")).toBeNull();
  });
});
