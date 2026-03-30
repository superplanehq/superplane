import { describe, expect, it } from "vitest";
import type { CanvasNode } from "@/ui/CanvasPage";
import { getNodeIntegrationName, overlayIntegrationWarnings } from "./node-integrations";

describe("getNodeIntegrationName", () => {
  const availableIntegrations = [
    {
      name: "github",
      components: [{ name: "github.create_issue" }],
      triggers: [{ name: "github.on_push" }],
    },
  ];

  it("finds the integration for component nodes", () => {
    expect(
      getNodeIntegrationName(
        {
          type: "TYPE_COMPONENT",
          component: { name: "github.create_issue" },
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
          trigger: { name: "github.on_push" },
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
          widget: { name: "group" },
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
