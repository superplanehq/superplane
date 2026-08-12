import { describe, expect, it } from "vitest";

import { resolveWorkOrderCreatorDisplay } from "./workOrderCreator";

const passthroughResolveUser = (userId: string | undefined, name?: string) =>
  userId ? { id: userId, name: name ?? "Member", initials: (name ?? "M").slice(0, 2).toUpperCase() } : null;

describe("resolveWorkOrderCreatorDisplay", () => {
  it("returns a member display when the creator is a user", () => {
    const display = resolveWorkOrderCreatorDisplay({ user: { id: "user-1", name: "Alice Smith" } }, passthroughResolveUser);
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
