import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ComponentsComponent, ComponentsNode } from "@/api-client";
import type { CustomFieldRenderer } from "./mappers/types";
import * as mappers from "./mappers";
import { createSafeCustomFieldRenderer } from "./mappers/safeMappers";
import { prepareComponentBaseNode, prepareTriggerNode } from "./lib/canvas-node-preparation";
import { renderCanvasNodeCustomField } from "./lib/render-canvas-node-custom-field";

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

function makeTriggerNode(overrides: Partial<ComponentsNode> = {}): ComponentsNode {
  return {
    id: "trigger-1",
    name: "Incoming Event",
    type: "TYPE_TRIGGER",
    position: { x: 0, y: 0 },
    trigger: {
      name: "webhook",
    },
    configuration: {},
    ...overrides,
  } as ComponentsNode;
}

describe("canvas node preparation resilience", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("returns a fallback canvas node when component preparation fails", () => {
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
      canvasId: "canvas-1",
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

    const result = renderCanvasNodeCustomField({
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

  it("keeps trigger error and warning precedence on node state only", () => {
    vi.spyOn(mappers, "getTriggerRenderer").mockReturnValue({
      getTriggerProps: () => ({
        title: "Webhook",
        iconSlug: "bolt",
        metadata: [],
        error: "renderer error",
        warning: "renderer warning",
      }),
      getRootEventValues: () => ({}),
      getTitleAndSubtitle: () => ({ title: "Event", subtitle: "" }),
    });

    const result = prepareTriggerNode(
      makeTriggerNode(),
      [{ name: "webhook", label: "Webhook", icon: "bolt" }] as never,
      {},
    );

    const triggerData = result.data as { trigger: { error?: string; warning?: string } };

    expect(triggerData.trigger.error).toBeUndefined();
    expect(triggerData.trigger.warning).toBeUndefined();
  });
});
