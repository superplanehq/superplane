import { describe, expect, it } from "vitest";

import { onRunTitle, onRunTriggerRenderer } from "./on_run";

describe("onRunTriggerRenderer", () => {
  it("builds a title from the calling app name", () => {
    expect(onRunTitle({ app: { name: "Billing" } })).toBe("Run from Billing");
    expect(onRunTitle(undefined)).toBe("App run");
  });

  it("maps root event values from app and payload", () => {
    expect(
      onRunTriggerRenderer.getRootEventValues({
        event: {
          id: "event-1",
          createdAt: "2026-07-15T12:00:00.000Z",
          nodeId: "on-run",
          type: "app.invocation",
          data: {
            app: { name: "Billing" },
            payload: {
              message: "hello",
              count: 2,
            },
          },
        },
      }),
    ).toEqual({
      App: "Billing",
      message: "hello",
      count: "2",
      "Received at": new Date("2026-07-15T12:00:00.000Z").toLocaleString(),
    });
  });

  it("shows parameter count on the trigger node", () => {
    const props = onRunTriggerRenderer.getTriggerProps({
      node: {
        id: "on-run",
        name: "Handle run",
        componentName: "onRun",
        isCollapsed: false,
        configuration: {
          parameters: [{ name: "message" }, { name: "count" }],
        },
      },
      definition: {
        name: "onRun",
        label: "On Run",
        description: "",
        icon: "play",
        color: "gray",
      },
      lastEvent: undefined,
    });

    expect(props.metadata).toEqual([{ icon: "list", label: "2 parameters" }]);
  });
});
