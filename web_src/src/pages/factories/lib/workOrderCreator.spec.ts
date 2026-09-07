import { describe, expect, it } from "vitest";

import { resolveWorkOrderCreatorDisplay, workOrderOwnerDisplay } from "./workOrderCreator";

const passthroughResolveUser = (userId: string | undefined, name?: string) =>
  userId ? { id: userId, name: name ?? "Member", initials: (name ?? "M").slice(0, 2).toUpperCase() } : null;

describe("resolveWorkOrderCreatorDisplay", () => {
  it("returns a member display when the creator is a user", () => {
    const display = resolveWorkOrderCreatorDisplay(
      { user: { id: "user-1", name: "Alice Smith" } },
      passthroughResolveUser,
    );
    expect(display).toMatchObject({ id: "user-1", name: "Alice Smith" });
  });

  it("falls back to an automation display when only the automation branch is set", () => {
    const display = resolveWorkOrderCreatorDisplay(
      { automation: { nodeId: "node-1", nodeName: "Release Gate", appId: "app-1", appName: "Release" } },
      passthroughResolveUser,
    );
    expect(display).toMatchObject({ id: "node-1", name: "Release Gate", initials: "RG" });
  });

  it("uses the app name when the automation has no node name", () => {
    const display = resolveWorkOrderCreatorDisplay(
      { automation: { appId: "app-1", appName: "Release" } },
      passthroughResolveUser,
    );
    expect(display).toMatchObject({ id: "app-1", name: "Release", initials: "R" });
  });

  it("returns null when neither branch carries usable identity", () => {
    expect(resolveWorkOrderCreatorDisplay(undefined, passthroughResolveUser)).toBeNull();
    expect(resolveWorkOrderCreatorDisplay({}, passthroughResolveUser)).toBeNull();
    expect(resolveWorkOrderCreatorDisplay({ automation: {} }, passthroughResolveUser)).toBeNull();
  });
});

const FALLBACK_OWNER = { id: "fallback", name: "Fallback User", initials: "FU", avatarUrl: "/fallback.jpg" };

describe("workOrderOwnerDisplay", () => {
  it("uses the automation display when createdBy is an automation", () => {
    expect(
      workOrderOwnerDisplay(
        { createdBy: { automation: { appId: "app-1", appName: "Refund Planner" } } },
        FALLBACK_OWNER,
      ),
    ).toMatchObject({ id: "app-1", name: "Refund Planner" });
  });

  it("uses the user display and copies the fallback avatar for the same id", () => {
    expect(workOrderOwnerDisplay({ createdBy: { user: { id: "fallback", name: "Ada" } } }, FALLBACK_OWNER)).toEqual({
      id: "fallback",
      name: "Ada",
      initials: "A",
      avatarUrl: "/fallback.jpg",
    });
  });

  it("returns the fallback when createdBy is empty", () => {
    expect(workOrderOwnerDisplay({}, FALLBACK_OWNER)).toBe(FALLBACK_OWNER);
  });

  it("resolves the user's avatar from the org members list when resolveUser finds one", () => {
    const resolveUser = (userId: string | undefined, name?: string) =>
      userId
        ? { id: userId, name: name ?? "Member", initials: "M", avatarUrl: "https://example.com/avatar.jpg" }
        : null;

    expect(
      workOrderOwnerDisplay(
        { createdBy: { user: { id: "user-1", name: "Ada" } } },
        FALLBACK_OWNER,
        resolveUser,
      ),
    ).toEqual({
      id: "user-1",
      name: "Ada",
      initials: "M",
      avatarUrl: "https://example.com/avatar.jpg",
    });
  });

  it("falls back to initials when resolveUser finds no avatar for the user", () => {
    const resolveUser = (userId: string | undefined, name?: string) =>
      userId ? { id: userId, name: name ?? "Member", initials: "AD" } : null;

    expect(
      workOrderOwnerDisplay(
        { createdBy: { user: { id: "user-1", name: "Ada" } } },
        FALLBACK_OWNER,
        resolveUser,
      ),
    ).toEqual({
      id: "user-1",
      name: "Ada",
      initials: "AD",
    });
  });
});
