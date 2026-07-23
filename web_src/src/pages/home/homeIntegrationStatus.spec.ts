import { describe, expect, it } from "vitest";

import { resolveHomeIntegrationStatus } from "./homeIntegrationStatus";
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
