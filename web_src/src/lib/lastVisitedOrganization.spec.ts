import { describe, expect, it, beforeEach } from "vitest";

import {
  LAST_VISITED_ORGANIZATION_STORAGE_KEY,
  pickAutoRedirectOrganization,
  readLastVisitedOrganization,
  recordLastVisitedOrganization,
} from "./lastVisitedOrganization";

describe("lastVisitedOrganization", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("returns null when nothing was recorded", () => {
    expect(readLastVisitedOrganization("account-1")).toBeNull();
  });

  it("records and reads the last visited organization per account", () => {
    recordLastVisitedOrganization("account-1", "org-a");
    recordLastVisitedOrganization("account-2", "org-b");

    expect(readLastVisitedOrganization("account-1")).toBe("org-a");
    expect(readLastVisitedOrganization("account-2")).toBe("org-b");
  });

  it("overwrites the previous organization for the same account", () => {
    recordLastVisitedOrganization("account-1", "org-a");
    recordLastVisitedOrganization("account-1", "org-b");

    expect(readLastVisitedOrganization("account-1")).toBe("org-b");
  });

  it("ignores malformed stored values", () => {
    window.localStorage.setItem(LAST_VISITED_ORGANIZATION_STORAGE_KEY, "not-json");
    expect(readLastVisitedOrganization("account-1")).toBeNull();

    window.localStorage.setItem(LAST_VISITED_ORGANIZATION_STORAGE_KEY, JSON.stringify(["org-a"]));
    expect(readLastVisitedOrganization("account-1")).toBeNull();

    window.localStorage.setItem(LAST_VISITED_ORGANIZATION_STORAGE_KEY, JSON.stringify({ "account-1": 42 }));
    expect(readLastVisitedOrganization("account-1")).toBeNull();
  });

  it("ignores empty account ids", () => {
    recordLastVisitedOrganization("", "org-a");
    expect(readLastVisitedOrganization("")).toBeNull();
  });
});

describe("pickAutoRedirectOrganization", () => {
  it("returns the only organization even without a last visited entry", () => {
    expect(pickAutoRedirectOrganization([{ id: "org-a" }], null)).toBe("org-a");
  });

  it("returns the last visited organization when the account still belongs to it", () => {
    const organizations = [{ id: "org-a" }, { id: "org-b" }];
    expect(pickAutoRedirectOrganization(organizations, "org-b")).toBe("org-b");
  });

  it("ignores a last visited organization the account no longer belongs to", () => {
    const organizations = [{ id: "org-a" }, { id: "org-b" }];
    expect(pickAutoRedirectOrganization(organizations, "org-gone")).toBeNull();
  });

  it("returns null with multiple organizations and no last visited entry", () => {
    expect(pickAutoRedirectOrganization([{ id: "org-a" }, { id: "org-b" }], null)).toBeNull();
  });

  it("returns null when there are no organizations", () => {
    expect(pickAutoRedirectOrganization([], "org-a")).toBeNull();
  });
});
