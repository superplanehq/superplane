import { describe, expect, it } from "vitest";

import type { SuperplaneUsersUser } from "@/api-client";

import {
  buildOrgUserDisplayMap,
  getOrgUserDisplayFromUser,
  getUserInitials,
  resolveOrgUserDisplay,
} from "./orgUserDisplay";

describe("orgUserDisplay", () => {
  it("derives initials from display name", () => {
    expect(getUserInitials("Alex Reviewer")).toBe("AR");
  });

  it("maps list users response fields to display data", () => {
    const user: SuperplaneUsersUser = {
      metadata: { id: "user-1", email: "alex@example.com" },
      spec: { displayName: "Alex Reviewer" },
      status: {
        accountProviders: [{ avatarUrl: "https://example.com/alex.png" }],
      },
    };

    expect(getOrgUserDisplayFromUser(user)).toEqual({
      id: "user-1",
      name: "Alex Reviewer",
      initials: "AR",
      avatarUrl: "https://example.com/alex.png",
    });
  });

  it("falls back to order-provided names when a user is missing from the roster", () => {
    const usersById = buildOrgUserDisplayMap([]);

    expect(resolveOrgUserDisplay(usersById, "missing-user", "Jamie Operator")).toEqual({
      id: "missing-user",
      name: "Jamie Operator",
      initials: "JO",
    });
  });
});
