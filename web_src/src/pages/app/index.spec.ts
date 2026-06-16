import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ActionsAction, SuperplaneComponentsNode } from "@/api-client";
import { makeCanvas, makeComponentsNode } from "@/test/factories";
import type { CustomFieldRenderer } from "./mappers/types";
import * as mappers from "./mappers";
import { createSafeCustomFieldRenderer } from "./mappers/safeMappers";
import { prepareComponentBaseNode, prepareTriggerNode } from "./lib/canvas-node-preparation";
import { renderCanvasNodeCustomField } from "./lib/render-canvas-node-custom-field";
import { getWorkflowSaveSignature, mapTriggerEventToSidebarEvent } from "./utils";

type FallbackComponentData = {
  renderFallback?: {
    source: string;
    message: string;
  };
  component: {
    error?: string;
    emptyStateProps?: {
      title?: string;
      purpose?: string;
    };
  };
};

function makeNode(overrides: Partial<SuperplaneComponentsNode> = {}): SuperplaneComponentsNode {
  return makeComponentsNode({
    id: "node-1",
    name: "Broken Component",
    type: "TYPE_ACTION",
    position: { x: 10, y: 20 },
    component: "approval",
    configuration: {},
    ...overrides,
  });
}

function makeComponent(overrides: Partial<ActionsAction> = {}): ActionsAction {
  return {
    name: "approval",
    label: "Approval",
    icon: "hand",
    color: "orange",
    outputChannels: [{ name: "default" }],
    ...overrides,
  } as ActionsAction;
}

function makeTriggerNode(overrides: Partial<SuperplaneComponentsNode> = {}): SuperplaneComponentsNode {
  return makeComponentsNode({
    id: "trigger-1",
    name: "Incoming Event",
    type: "TYPE_TRIGGER",
    position: { x: 0, y: 0 },
    component: "webhook",
    configuration: {},
    ...overrides,
  });
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
    });

    const fallbackData = result.data as unknown as FallbackComponentData;

    expect(fallbackData.renderFallback).toEqual({
      source: "mapper",
      message: "Can't display",
    });
    expect(fallbackData.component.error).toBeUndefined();
    expect(fallbackData.component.emptyStateProps?.title).toBe("Can't display");
    expect(fallbackData.component.emptyStateProps?.purpose).toBe("fallback");
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
      "live",
      { canvasId: "canvas-1", openModal: () => {} },
    );

    const triggerData = result.data as { trigger: { error?: string; warning?: string } };

    expect(triggerData.trigger.error).toBeUndefined();
    expect(triggerData.trigger.warning).toBeUndefined();
  });

  it("does not prepare trigger actions when no modal opener is available", () => {
    const getTriggerProps = vi.fn(() => ({
      title: "Webhook",
      iconSlug: "bolt",
      metadata: [],
    }));
    vi.spyOn(mappers, "getTriggerRenderer").mockReturnValue({
      getTriggerProps,
      getRootEventValues: () => ({}),
      getTitleAndSubtitle: () => ({ title: "Event", subtitle: "" }),
    });

    prepareTriggerNode(makeTriggerNode(), [{ name: "webhook", label: "Webhook", icon: "bolt" }] as never, {}, "live", {
      canvasId: "canvas-1",
    });

    expect(getTriggerProps).toHaveBeenCalledWith(expect.objectContaining({ actions: undefined }));
  });

  it("uses componentName when rendering trigger event details", () => {
    const node = makeComponentsNode({
      id: "trigger-1",
      name: "github.onCheckRun",
      type: "TYPE_TRIGGER",
      component: undefined,
      componentName: "github.onCheckRun",
    } as Partial<SuperplaneComponentsNode> & { componentName: string });

    const event = {
      id: "event-1",
      nodeId: "trigger-1",
      createdAt: new Date().toISOString(),
      data: {
        type: "github.checkRun",
        data: {
          action: "completed",
          check_run: {
            name: "DCO",
            status: "completed",
            conclusion: "success",
            head_sha: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
            details_url: "https://github.com/apps/dco/checks/1",
            app: {
              name: "DCO",
              client_id: "lv1.raw-field-that-should-not-render",
            },
            check_suite: { head_branch: "feature/check-runs" },
            pull_requests: [{ number: 42 }],
          },
          repository: {
            full_name: "acme/snaketoy",
          },
        },
      },
    };

    const sidebarEvent = mapTriggerEventToSidebarEvent(event, node);
    const values = sidebarEvent.values || {};

    expect(sidebarEvent.title).toBe("DCO succeeded - d6f3c8a");
    expect(Object.keys(values)).toEqual([
      "Action",
      "Name",
      "Status",
      "Conclusion",
      "Branch",
      "SHA",
      "Pull request",
      "App",
      "Details URL",
      "Repository",
    ]);
    expect(values).toMatchObject({
      Name: "DCO",
      Status: "completed",
      Conclusion: "success",
      Branch: "feature/check-runs",
      "Pull request": "#42",
      App: "DCO",
      Repository: "acme/snaketoy",
    });
    expect(values).not.toHaveProperty("client_id");
  });
});

describe("getWorkflowSaveSignature", () => {
  it("treats integration refs with the same id as the same saved draft even when only the display name differs", () => {
    const workflowWithIntegrationName = makeCanvas({
      spec: {
        nodes: [
          makeNode({
            integration: {
              id: "integration-1",
              name: "github-1",
            },
          }),
        ],
        edges: [],
      },
    });

    const workflowWithPersistedIntegrationShape = makeCanvas({
      spec: {
        nodes: [
          makeNode({
            integration: {
              id: "integration-1",
            },
          }),
        ],
        edges: [],
      },
    });

    expect(getWorkflowSaveSignature(workflowWithIntegrationName)).toBe(
      getWorkflowSaveSignature(workflowWithPersistedIntegrationShape),
    );
  });
});
