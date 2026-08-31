import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { CanvasesCanvas } from "@/api-client";
import type { FactoryConfigureActions } from "@/pages/app";

import { useFactoryAppCanvasEditActions } from "./useFactoryAppCanvasEditActions";

function implementCanvas(): CanvasesCanvas {
  return {
    metadata: { id: "app-1", name: "Implement" },
    spec: {
      edges: [],
      nodes: [
        { id: "onrun-implement", name: "Start", type: "TYPE_TRIGGER", component: "onRun", configuration: {} },
        {
          id: "implementation-agent-no-issue",
          name: "Agent - Implement from order description",
          type: "TYPE_ACTION",
          component: "runnerClaudeCode",
          configuration: {
            model: "sonnet",
            credentials: { source: "integration", integration: { name: "claude-prod" } },
            environment: [
              { name: "REPO", value: "acme/web" },
              { name: "BASE", value: "main" },
            ],
          },
        },
        {
          id: "create-pr",
          name: "Create Pull Request",
          type: "TYPE_ACTION",
          component: "github.createPullRequest",
          configuration: { repository: "acme/web", base: "main" },
          integration: { id: "gh-1", name: "github-prod" },
        },
      ],
    },
  };
}

function baseOptions(overrides: Partial<Parameters<typeof useFactoryAppCanvasEditActions>[0]> = {}) {
  return {
    organizationId: "org-1",
    factoryKey: "SP",
    appId: "app-1",
    from: null,
    lineId: null,
    orderNumber: null,
    runId: null,
    isConfigure: true,
    agentOpen: false,
    componentsOpen: false,
    canvas: implementCanvas(),
    canUpdateCanvas: true,
    configureActionsRef: { current: { applyDraftSpec: vi.fn() } as unknown as FactoryConfigureActions },
    setSearchParams: vi.fn(),
    navigate: vi.fn(),
    ...overrides,
  };
}

describe("useFactoryAppCanvasEditActions reset to factory defaults", () => {
  it("is unavailable when no bundled template matches the canvas", () => {
    const { result } = renderHook(() =>
      useFactoryAppCanvasEditActions(
        baseOptions({ canvas: { metadata: { id: "app-1" }, spec: { nodes: [{ id: "unmatched" }], edges: [] } } }),
      ),
    );

    expect(result.current.resetAvailable).toBe(false);
  });

  it("is unavailable outside configure mode", () => {
    const { result } = renderHook(() => useFactoryAppCanvasEditActions(baseOptions({ isConfigure: false })));

    expect(result.current.resetAvailable).toBe(false);
  });

  it("is unavailable without update permission", () => {
    const { result } = renderHook(() => useFactoryAppCanvasEditActions(baseOptions({ canUpdateCanvas: false })));

    expect(result.current.resetAvailable).toBe(false);
  });

  it("opens the confirm dialog when reset is available", () => {
    const { result } = renderHook(() => useFactoryAppCanvasEditActions(baseOptions()));

    expect(result.current.resetAvailable).toBe(true);
    expect(result.current.resetConfirmOpen).toBe(false);

    act(() => {
      result.current.handleOpenResetConfirm();
    });

    expect(result.current.resetConfirmOpen).toBe(true);
  });

  it("does not open the confirm dialog when reset is unavailable", () => {
    const { result } = renderHook(() => useFactoryAppCanvasEditActions(baseOptions({ canUpdateCanvas: false })));

    act(() => {
      result.current.handleOpenResetConfirm();
    });

    expect(result.current.resetConfirmOpen).toBe(false);
  });

  it("materializes the app's own wiring and applies it as an unsaved draft", () => {
    const applyDraftSpec = vi.fn();
    const configureActionsRef = { current: { applyDraftSpec } as unknown as FactoryConfigureActions };
    const { result } = renderHook(() => useFactoryAppCanvasEditActions(baseOptions({ configureActionsRef })));

    act(() => {
      result.current.handleOpenResetConfirm();
      result.current.handleResetToFactoryDefaults();
    });

    // Closes the confirm dialog and never calls save — the caller still has
    // to click Save to persist the draft.
    expect(result.current.resetConfirmOpen).toBe(false);
    expect(applyDraftSpec).toHaveBeenCalledTimes(1);

    type MaterializedNode = {
      id: string;
      configuration?: Record<string, unknown>;
      integration?: { id?: string; name?: string };
    };
    const spec = applyDraftSpec.mock.calls[0][0] as { nodes: MaterializedNode[] };
    const nodeById = new Map(spec.nodes.map((node) => [node.id, node]));
    expect(nodeById.get("create-pr")?.configuration?.repository).toBe("acme/web");
    expect(nodeById.get("create-pr")?.integration).toEqual({ id: "gh-1", name: "github-prod" });
    expect(nodeById.get("implementation-agent-no-issue")?.configuration?.credentials).toEqual({
      source: "integration",
      integration: { name: "claude-prod" },
    });
  });

  it("does nothing when reset is unavailable", () => {
    const applyDraftSpec = vi.fn();
    const configureActionsRef = { current: { applyDraftSpec } as unknown as FactoryConfigureActions };
    const { result } = renderHook(() =>
      useFactoryAppCanvasEditActions(baseOptions({ canUpdateCanvas: false, configureActionsRef })),
    );

    act(() => {
      result.current.handleResetToFactoryDefaults();
    });

    expect(applyDraftSpec).not.toHaveBeenCalled();
  });
});
