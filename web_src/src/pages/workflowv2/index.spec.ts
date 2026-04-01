import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ComponentsComponent, ComponentsNode } from "@/api-client";
import type { CustomFieldRenderer } from "./mappers/types";
import * as mappers from "./mappers";
import { prepareComponentBaseNode, renderWorkflowNodeCustomField } from "./index";

type FallbackComponentData = {
  renderFallback?: {
    source: string;
    message: string;
  };
  component: {
    error?: string;
    emptyStateProps?: {
      title?: string;
    };
  };
};

function makeNode(overrides: Partial<ComponentsNode> = {}): ComponentsNode {
  return {
    id: "node-1",
    name: "Broken Component",
    type: "TYPE_COMPONENT",
    position: { x: 10, y: 20 },
    component: {
      name: "approval",
    },
    configuration: {},
    ...overrides,
  } as ComponentsNode;
}

function makeComponent(overrides: Partial<ComponentsComponent> = {}): ComponentsComponent {
  return {
    name: "approval",
    label: "Approval",
    icon: "hand",
    color: "orange",
    outputChannels: [{ name: "default" }],
    ...overrides,
  } as ComponentsComponent;
}

describe("workflow node preparation resilience", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("returns a fallback canvas node when component preparation fails", () => {
    vi.spyOn(mappers, "getComponentAdditionalDataBuilder").mockReturnValue({
      buildAdditionalData: () => {
        throw new Error("builder failed");
      },
    });
    vi.spyOn(mappers, "getComponentBaseMapper").mockReturnValue({
      props: () => {
        throw new Error("mapper failed");
      },
      subtitle: () => "",
      getExecutionDetails: () => ({}),
    });

    const result = prepareComponentBaseNode(
      [makeNode()],
      makeNode(),
      [makeComponent()],
      {},
      {},
      "canvas-1",
      new QueryClient(),
      "org-1",
    );

    const fallbackData = result.data as unknown as FallbackComponentData;

    expect(fallbackData.renderFallback).toEqual({
      source: "mapper",
      message: "Unavailable",
    });
    expect(fallbackData.component.error).toBeUndefined();
    expect(fallbackData.component.emptyStateProps?.title).toBe("Unavailable");
  });

  it("returns null when a custom field renderer throws so sidebar rendering stays alive", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const renderer: CustomFieldRenderer = {
      render: () => {
        throw new Error("custom field failed");
      },
    };

    const result = renderWorkflowNodeCustomField({
      renderer,
      node: makeNode(),
      nodeId: "node-1",
    });

    expect(result).toBeNull();
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Failed to render custom field for node "node-1"'),
      expect.any(Error),
    );
    consoleSpy.mockRestore();
  });
});
