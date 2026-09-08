import { describe, expect, it } from "vitest";

import {
  requiredIntegrationsReady,
  resolveHomeIntegrationStatus,
  syncSelectionsWithInstances,
} from "./homeIntegrationStatus";
import type { OrganizationsIntegration } from "@/api-client";

function instance(
  id: string,
  state: "ready" | "pending" | "error",
  integrationName = "github",
): OrganizationsIntegration {
  return {
    metadata: { id, name: id, integrationName },
    status: { state },
  } as OrganizationsIntegration;
}

describe("resolveHomeIntegrationStatus", () => {
  it("returns Not connected when there are no instances", () => {
    expect(resolveHomeIntegrationStatus({ name: "github", allInstances: [], readyInstances: [] })).toEqual({
      kind: "none",
      label: "Not connected",
    });
  });

  it("returns Connected when any ready instance exists", () => {
    expect(
      resolveHomeIntegrationStatus({
        name: "github",
        allInstances: [instance("a", "ready"), instance("b", "pending")],
        readyInstances: [instance("a", "ready")],
      }),
    ).toEqual({ kind: "ready", label: "Connected" });
  });

  it("returns Pending for the selected instance even when another is ready", () => {
    expect(
      resolveHomeIntegrationStatus(
        {
          name: "github",
          allInstances: [instance("old", "ready"), instance("new", "pending")],
          readyInstances: [instance("old", "ready")],
        },
        "new",
      ),
    ).toEqual({ kind: "pending", label: "Pending", configureId: "new" });
  });

  it("returns Pending for pending instances when none are ready", () => {
    expect(
      resolveHomeIntegrationStatus({
        name: "github",
        allInstances: [instance("p", "pending")],
        readyInstances: [],
      }),
    ).toEqual({ kind: "pending", label: "Pending", configureId: "p" });
  });

  it("returns Error for errored instances when none are ready or pending", () => {
    expect(
      resolveHomeIntegrationStatus({
        name: "github",
        allInstances: [instance("e", "error")],
        readyInstances: [],
      }),
    ).toEqual({ kind: "error", label: "Error", configureId: "e" });
  });
});

describe("requiredIntegrationsReady", () => {
  it("requires ready selections, not just selected ids", () => {
    expect(requiredIntegrationsReady(["github"], { github: { id: "new", name: "new", ready: false } })).toBe(false);
    expect(requiredIntegrationsReady(["github"], { github: { id: "new", name: "new", ready: true } })).toBe(true);
  });
});

describe("syncSelectionsWithInstances", () => {
  it("auto-selects the first ready instance when none is selected", () => {
    const data = [
      {
        name: "github",
        allInstances: [instance("old", "ready"), instance("new", "ready")],
        readyInstances: [instance("old", "ready"), instance("new", "ready")],
      },
    ];
    expect(syncSelectionsWithInstances(data, {})).toEqual({
      github: { id: "old", name: "old", ready: true },
    });
  });

  it("keeps a preferred instance even when an older ready instance exists", () => {
    const data = [
      {
        name: "github",
        allInstances: [instance("old", "ready"), instance("new", "ready")],
        readyInstances: [instance("old", "ready"), instance("new", "ready")],
      },
    ];
    expect(
      syncSelectionsWithInstances(data, { github: { id: "old", name: "old", ready: true } }, { github: "new" }),
    ).toEqual({
      github: { id: "new", name: "new", ready: true },
    });
  });

  it("keeps a preferred pending instance instead of clearing it", () => {
    const data = [
      {
        name: "github",
        allInstances: [instance("old", "ready"), instance("new", "pending")],
        readyInstances: [instance("old", "ready")],
      },
    ];
    expect(
      syncSelectionsWithInstances(data, { github: { id: "old", name: "old", ready: true } }, { github: "new" }),
    ).toEqual({
      github: { id: "new", name: "new", ready: false },
    });
  });

  // Onboarding lists github as a manual selection: the wizard must not adopt
  // an existing organization connection, or the repository list would come
  // from an account the user did not pick.
  it("never auto-selects a manual-selection integration", () => {
    const data = [
      {
        name: "github",
        allInstances: [instance("old", "ready")],
        readyInstances: [instance("old", "ready")],
      },
    ];
    expect(syncSelectionsWithInstances(data, {}, {}, ["github"])).toBeNull();
  });

  it("keeps a saved selection for a manual-selection integration", () => {
    const data = [
      {
        name: "github",
        allInstances: [instance("saved", "ready")],
        readyInstances: [instance("saved", "ready")],
      },
    ];
    expect(
      syncSelectionsWithInstances(data, { github: { id: "saved", name: "saved", ready: false } }, {}, ["github"]),
    ).toEqual({
      github: { id: "saved", name: "saved", ready: true },
    });
  });

  it("applies a preferred instance for a manual-selection integration", () => {
    const data = [
      {
        name: "github",
        allInstances: [instance("old", "ready"), instance("picked", "ready")],
        readyInstances: [instance("old", "ready"), instance("picked", "ready")],
      },
    ];
    expect(syncSelectionsWithInstances(data, {}, { github: "picked" }, ["github"])).toEqual({
      github: { id: "picked", name: "picked", ready: true },
    });
  });
});
