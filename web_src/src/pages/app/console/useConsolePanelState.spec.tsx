import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import { useConsolePanelState } from "./useConsolePanelState";

const LAYOUT: ConsoleLayoutItem[] = [{ i: "deploy", x: 0, y: 0, w: 12, h: 6, minW: 2, minH: 2 }];

describe("useConsolePanelState — legacy `node` panel migration", () => {
  it("rewrites the type to `nodes` when the merged card emits a nodes-list body", () => {
    const legacyPanel: ConsolePanel = {
      id: "deploy",
      type: "node",
      content: { title: "Deploy", node: "deploy-prod", showRun: true },
    };
    const onChange = vi.fn();
    const onEffectiveChange = vi.fn();

    vi.useFakeTimers();
    try {
      const { result } = renderHook(() => useConsolePanelState([legacyPanel], LAYOUT, onChange, onEffectiveChange));

      act(() => {
        result.current.handlePanelContentChange("deploy", {
          title: "Deploy",
          nodes: [{ node: "deploy-prod", showRun: true }],
        });
      });

      const migrated = result.current.localPanels[0];
      expect(migrated.type).toBe("nodes");
      expect(migrated.content).toEqual({
        title: "Deploy",
        nodes: [{ node: "deploy-prod", showRun: true }],
      });

      act(() => {
        vi.runAllTimers();
      });

      expect(onChange).toHaveBeenCalledTimes(1);
      const [saved] = onChange.mock.calls[0];
      expect(saved.panels[0].type).toBe("nodes");
    } finally {
      vi.useRealTimers();
    }
  });

  it("leaves the panel type untouched when content shape does not match", () => {
    const legacyPanel: ConsolePanel = {
      id: "deploy",
      type: "node",
      content: { title: "Deploy", node: "deploy-prod" },
    };
    const onChange = vi.fn();

    vi.useFakeTimers();
    try {
      const { result } = renderHook(() => useConsolePanelState([legacyPanel], LAYOUT, onChange));

      act(() => {
        result.current.handlePanelContentChange("deploy", { title: "Renamed", node: "deploy-prod" });
      });

      expect(result.current.localPanels[0].type).toBe("node");
    } finally {
      vi.useRealTimers();
    }
  });
});
