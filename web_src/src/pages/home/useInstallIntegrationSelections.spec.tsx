import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import type { OrganizationsIntegration } from "@/api-client";

import type { IntegrationInstanceSummary } from "./homeIntegrationStatus";
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

  it("selects the integration that completed setup after the connection list loads", () => {
    const onSelectionsChange = vi.fn();
    const returnedOpenRouter: OrganizationsIntegration = {
      metadata: { id: "returned-openrouter", name: "openrouter-4", integrationName: "openrouter" },
      status: { state: "ready" },
    };
    const { rerender } = renderHook(
      ({ loading, integrationData }: { loading: boolean; integrationData: IntegrationInstanceSummary[] }) =>
        useInstallIntegrationSelections({
          integrationData,
          selections: {},
          onSelectionsChange,
          manualSelectionNames: ["openrouter"],
          loading,
          initialPreferredIntegrationId: "returned-openrouter",
        }),
      {
        initialProps: {
          loading: true,
          integrationData: [
            {
              name: "openrouter",
              allInstances: [] as OrganizationsIntegration[],
              readyInstances: [] as OrganizationsIntegration[],
            },
          ],
        },
      },
    );

    expect(onSelectionsChange).not.toHaveBeenCalled();

    rerender({
      loading: false,
      integrationData: [
        {
          name: "openrouter",
          allInstances: [returnedOpenRouter],
          readyInstances: [returnedOpenRouter],
        },
      ],
    });

    expect(onSelectionsChange).toHaveBeenCalledWith({
      openrouter: { id: "returned-openrouter", name: "openrouter-4", ready: true },
    });
  });
});
