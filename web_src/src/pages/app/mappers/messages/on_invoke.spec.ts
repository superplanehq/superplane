import { describe, expect, it } from "vitest";

import { invokeTitle, onInvokeTriggerRenderer } from "./on_invoke";

describe("onInvokeTriggerRenderer", () => {
  it("builds a title from the calling app name", () => {
    expect(invokeTitle({ app: { name: "Billing" } })).toBe("Invoked from Billing");
    expect(invokeTitle(undefined)).toBe("App invoked");
  });

  it("maps root event values from app and payload", () => {
    expect(
      onInvokeTriggerRenderer.getRootEventValues({
        event: {
          id: "event-1",
          createdAt: "2026-07-15T12:00:00.000Z",
          nodeId: "on-invoke",
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
    const props = onInvokeTriggerRenderer.getTriggerProps({
      node: {
        id: "on-invoke",
        name: "Handle invoke",
        componentName: "onInvoke",
        isCollapsed: false,
        configuration: {
          parameters: [{ name: "message" }, { name: "count" }],
        },
      },
      definition: {
        name: "onInvoke",
        label: "On Invoke",
        description: "",
        icon: "play",
        color: "gray",
      },
      lastEvent: undefined,
    });

    expect(props.metadata).toEqual([{ icon: "list", label: "2 parameters" }]);
  });
});
