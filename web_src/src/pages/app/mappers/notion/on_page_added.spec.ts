import { describe, expect, it } from "vitest";
import type { ComponentDefinition, EventInfo, NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onPageAddedTriggerRenderer } from "./on_page_added";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-03-26T19:29:35Z").toISOString(),
    nodeId: "node-1",
    type: "notion.page",
    data,
  };
}

const pageEvent = {
  meta: { event: "page.created" },
  data: {
    id: "8a1e6d2f-8b0e-4e9b-9a3a-3a6b2f9c9d10",
    url: "https://www.notion.so/Fix-payment-retries",
    title: "Fix payment retries",
    content: "Retries fail silently after the third attempt.",
  },
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Page Added",
    componentName: "notion.onPageAdded",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildTriggerContext(overrides?: {
  node?: Partial<NodeInfo>;
  lastEvent?: EventInfo;
  definition?: Partial<ComponentDefinition>;
}): TriggerRendererContext {
  return {
    node: buildNode(overrides?.node),
    definition: {
      name: "notion.onPageAdded",
      label: "On Page Added",
      description: "",
      icon: "notion",
      color: "gray",
      ...overrides?.definition,
    },
    lastEvent: overrides?.lastEvent,
  } as TriggerRendererContext;
}

describe("onPageAddedTriggerRenderer.getTitleAndSubtitle", () => {
  it("builds the title from the page", () => {
    const context: TriggerEventContext = { event: event(pageEvent) };
    expect(onPageAddedTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Fix payment retries");
  });

  it("falls back to a generic title when the payload is empty", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onPageAddedTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Page");
  });

  it("does not throw when there is no event", () => {
    expect(() => onPageAddedTriggerRenderer.getTitleAndSubtitle({ event: undefined })).not.toThrow();
  });
});

describe("onPageAddedTriggerRenderer.getRootEventValues", () => {
  it("maps the page fields, including the content", () => {
    const values = onPageAddedTriggerRenderer.getRootEventValues({ event: event(pageEvent) });

    expect(values["Received At"]).toBeDefined();
    expect(values["Page"]).toBe("8a1e6d2f-8b0e-4e9b-9a3a-3a6b2f9c9d10");
    expect(values["Title"]).toBe("Fix payment retries");
    expect(values["Content"]).toBe("Retries fail silently after the third attempt.");
    expect(values["URL"]).toBe("https://www.notion.so/Fix-payment-retries");
  });

  it("falls back to dashes when the payload is empty", () => {
    const values = onPageAddedTriggerRenderer.getRootEventValues({ event: event({}) });

    expect(values["Page"]).toBe("-");
    expect(values["Title"]).toBe("-");
    expect(values["Content"]).toBe("-");
  });

  it("does not throw when there is no event", () => {
    expect(() => onPageAddedTriggerRenderer.getRootEventValues({ event: undefined })).not.toThrow();
  });
});

describe("onPageAddedTriggerRenderer.getTriggerProps", () => {
  it("renders the configured database", () => {
    const props = onPageAddedTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { database: "db-1" },
          metadata: { database: { id: "db-1", name: "Tasks" } },
        },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "database", label: "Tasks" });
  });

  it("falls back to the configured database id when metadata is missing", () => {
    const props = onPageAddedTriggerRenderer.getTriggerProps(
      buildTriggerContext({ node: { configuration: { database: "db-1" } } }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "database", label: "db-1" });
  });

  it("surfaces the last event when one exists", () => {
    const props = onPageAddedTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: { configuration: { database: "db-1" } },
        lastEvent: event(pageEvent),
      }),
    );

    expect(props.lastEventData?.title).toBe("Fix payment retries");
    expect(props.lastEventData?.state).toBe("triggered");
  });

  it("does not throw when configuration and metadata are undefined", () => {
    expect(() =>
      onPageAddedTriggerRenderer.getTriggerProps(
        buildTriggerContext({ node: { configuration: undefined, metadata: undefined } }),
      ),
    ).not.toThrow();
  });
});
