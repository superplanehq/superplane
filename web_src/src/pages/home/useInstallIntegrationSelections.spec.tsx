import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import type { OrganizationsIntegration } from "@/api-client";

import { useInstallIntegrationSelections } from "./useInstallIntegrationSelections";

const savedSelections = { github: { id: "saved-github", name: "saved-github", ready: false } };

describe("useInstallIntegrationSelections", () => {
  // A saved selection points at a connection the list has not returned yet.
  // A sync against the empty list would drop the saved answer for good.
  it("keeps a saved selection while the connection list loads", () => {
    const onSelectionsChange = vi.fn();

    renderHook(() =>
      useInstallIntegrationSelections({
        integrationData: [{ name: "github", allInstances: [], readyInstances: [] }],
        selections: savedSelections,
        onSelectionsChange,
        manualSelectionNames: ["github"],
        loading: true,
      }),
    );

    expect(onSelectionsChange).not.toHaveBeenCalled();
  });

  it("marks the saved selection ready once the connection list arrives", () => {
    const onSelectionsChange = vi.fn();
    const instance: OrganizationsIntegration = {
      metadata: { id: "saved-github", name: "acme-github" },
      status: { state: "ready" },
    };

    renderHook(() =>
      useInstallIntegrationSelections({
        integrationData: [{ name: "github", allInstances: [instance], readyInstances: [instance] }],
        selections: savedSelections,
        onSelectionsChange,
        manualSelectionNames: ["github"],
        loading: false,
      }),
    );

    expect(onSelectionsChange).toHaveBeenCalledWith({
      github: { id: "saved-github", name: "acme-github", ready: true },
    });
  });
});
