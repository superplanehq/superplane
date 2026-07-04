import { describe, expect, it } from "vitest";
import type { CanvasNode } from "@/ui/CanvasPage";
import {
  getNodeIntegrationName,
  overlayIntegrationWarnings,
  stripCanvasNodeSetupWarningsForRunsView,
} from "./node-integrations";

describe("getNodeIntegrationName", () => {
  const availableIntegrations = [
    {
      name: "github",
      capabilities: [
        { type: "TYPE_ACTION" as const, name: "github.create_issue" },
        { type: "TYPE_TRIGGER" as const, name: "github.on_push" },
      ],
    },
  ];

  it("finds the integration for component nodes", () => {
    expect(
      getNodeIntegrationName(
        {
          type: "TYPE_ACTION",
          component: "github.create_issue",
        },
        availableIntegrations,
      ),
    ).toBe("github");
  });

  it("finds the integration for trigger nodes", () => {
    expect(
      getNodeIntegrationName(
        {
          type: "TYPE_TRIGGER",
          component: "github.on_push",
        },
        availableIntegrations,
      ),
    ).toBe("github");
  });

  it("returns undefined when the node does not match an integration", () => {
    expect(
      getNodeIntegrationName(
        {
          type: "TYPE_WIDGET",
          component: "annotation",
        },
        availableIntegrations,
      ),
    ).toBeUndefined();
  });
});

describe("overlayIntegrationWarnings", () => {
  it("adds an error message to component nodes using non-ready integrations", () => {
    const nodes = [
      {
        id: "node-1",
        position: { x: 0, y: 0 },
        data: {
          component: {},
        },
      } as CanvasNode,
    ];

    const result = overlayIntegrationWarnings(
      nodes,
      [
        {
          metadata: { id: "integration-1" },
          status: { state: "error", stateDescription: "Connection failed" },
        },
      ],
      [
        {
          id: "node-1",
          integration: { id: "integration-1" },
        },
      ],
    );

    expect((result[0].data as { component: { error?: string } }).component.error).toBe(
      "Integration error: Connection failed",
    );
  });

  it("adds a readiness warning to trigger nodes", () => {
    const nodes = [
      {
        id: "node-2",
        position: { x: 0, y: 0 },
        data: {
          trigger: {},
        },
      } as CanvasNode,
    ];

    const result = overlayIntegrationWarnings(
      nodes,
      [
        {
          metadata: { id: "integration-2" },
          status: { state: "pending" },
        },
      ],
      [
        {
          id: "node-2",
          integration: { id: "integration-2" },
        },
      ],
    );

    expect((result[0].data as { trigger: { error?: string } }).trigger.error).toBe("Integration is pending");
  });

  it("does not overwrite an existing node error", () => {
    const nodes = [
      {
        id: "node-3",
        position: { x: 0, y: 0 },
        data: {
          component: {
            error: "Existing error",
          },
        },
      } as CanvasNode,
    ];

    const result = overlayIntegrationWarnings(
      nodes,
      [
        {
          metadata: { id: "integration-3" },
          status: { state: "error", stateDescription: "Connection failed" },
        },
      ],
      [
        {
          id: "node-3",
          integration: { id: "integration-3" },
        },
      ],
    );

    expect((result[0].data as { component: { error?: string } }).component.error).toBe("Existing error");
  });
});

describe("stripCanvasNodeSetupWarningsForRunsView", () => {
  it("removes component error and warning", () => {
    const nodes = [
      {
        id: "n1",
        position: { x: 0, y: 0 },
        data: {
          type: "component",
          label: "X",
          state: "pending",
          component: { title: "A", error: "Not ready", warning: "Fix me" },
        },
      } as CanvasNode,
    ];

    const result = stripCanvasNodeSetupWarningsForRunsView(nodes);
    const component = (result[0].data as { component: Record<string, unknown> }).component;
    expect(component.error).toBeUndefined();
    expect(component.warning).toBeUndefined();
    expect(component.title).toBe("A");
  });

  it("removes trigger error and warning", () => {
    const nodes = [
      {
        id: "n2",
        position: { x: 0, y: 0 },
        data: {
          type: "trigger",
          label: "T",
          state: "pending",
          trigger: { title: "B", error: "Bad", warning: "Hmm" },
        },
      } as CanvasNode,
    ];

    const result = stripCanvasNodeSetupWarningsForRunsView(nodes);
    const trigger = (result[0].data as { trigger: Record<string, unknown> }).trigger;
    expect(trigger.error).toBeUndefined();
    expect(trigger.warning).toBeUndefined();
  });

  it("removes composite error and warning", () => {
    const nodes = [
      {
        id: "n3",
        position: { x: 0, y: 0 },
        data: {
          type: "composite",
          label: "C",
          state: "pending",
          composite: { title: "Nested", error: "Bad", warning: "Hmm" },
        },
      } as CanvasNode,
    ];

    const result = stripCanvasNodeSetupWarningsForRunsView(nodes);
    const composite = (result[0].data as { composite: Record<string, unknown> }).composite;
    expect(composite.error).toBeUndefined();
    expect(composite.warning).toBeUndefined();
    expect(composite.title).toBe("Nested");
  });
});
