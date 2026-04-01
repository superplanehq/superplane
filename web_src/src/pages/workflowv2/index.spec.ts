import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ComponentsComponent, ComponentsNode } from "@/api-client";
import type { CustomFieldRenderer } from "./mappers/types";
import * as mappers from "./mappers";
import { createSafeCustomFieldRenderer } from "./mappers/safeMappers";
import { prepareComponentBaseNode } from "./lib/canvas-node-preparation";
import { renderWorkflowNodeCustomField } from "./lib/render-workflow-node-custom-field";

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

    const result = prepareComponentBaseNode({
      nodes: [makeNode()],
      node: makeNode(),
      components: [makeComponent()],
      nodeExecutionsMap: {},
      nodeQueueItemsMap: {},
      workflowId: "canvas-1",
      queryClient: new QueryClient(),
      organizationId: "org-1",
    });

    const fallbackData = result.data as unknown as FallbackComponentData;

    expect(fallbackData.renderFallback).toEqual({
      source: "mapper",
      message: "Can't display",
    });
    expect(fallbackData.component.error).toBeUndefined();
    expect(fallbackData.component.emptyStateProps?.title).toBe("Can't display");
  });

  it("returns null when a custom field renderer throws so sidebar rendering stays alive", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const renderer = createSafeCustomFieldRenderer(
      {
        render: () => {
          throw new Error("custom field failed");
        },
      } satisfies CustomFieldRenderer,
      "approval",
    );

    const result = renderWorkflowNodeCustomField({
      renderer,
      node: makeNode(),
    });

    expect(result).toBeNull();
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('Custom field renderer "approval" threw in render()'),
      expect.any(Error),
    );
    consoleSpy.mockRestore();
  });
});
