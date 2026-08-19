import { describe, expect, it } from "vitest";
import type { ComponentDefinition, EventInfo, NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onIssueLabelTriggerRenderer } from "./on_issue_label";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-03-26T19:29:35Z").toISOString(),
    nodeId: "node-1",
    type: "linear.issue",
    data,
  };
}

const labelEvent = {
  action: "update",
  type: "Issue",
  url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry",
  actor: { id: "u1", name: "John Doe", email: "john@example.com", type: "user" },
  data: {
    id: "2174add1",
    identifier: "ENG-142",
    title: "Deploy pipeline fails on retry",
    state: { id: "s1", name: "Todo", type: "unstarted" },
    team: { id: "t1", key: "ENG", name: "Engineering" },
    labels: [
      { id: "l1", name: "bug" },
      { id: "l2", name: "factory" },
    ],
  },
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Issue Label",
    componentName: "linear.onIssueLabel",
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
      name: "linear.onIssueLabel",
      label: "On Issue Label",
      description: "",
      icon: "linear",
      color: "indigo",
      ...overrides?.definition,
    },
    lastEvent: overrides?.lastEvent,
  } as TriggerRendererContext;
}

describe("onIssueLabelTriggerRenderer.getTitleAndSubtitle", () => {
  it("builds the title from the issue identifier and title", () => {
    const context: TriggerEventContext = { event: event(labelEvent) };
    expect(onIssueLabelTriggerRenderer.getTitleAndSubtitle(context).title).toBe(
      "ENG-142 · Deploy pipeline fails on retry",
    );
  });

  it("falls back to a generic title when the payload is empty", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onIssueLabelTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Issue");
  });

  it("does not throw when there is no event", () => {
    expect(() => onIssueLabelTriggerRenderer.getTitleAndSubtitle({ event: undefined })).not.toThrow();
  });
});

describe("onIssueLabelTriggerRenderer.getRootEventValues", () => {
  it("maps the issue fields, including the labels and a link to the issue", () => {
    const values = onIssueLabelTriggerRenderer.getRootEventValues({ event: event(labelEvent) });

    expect(values["Received At"]).toBeDefined();
    expect(values["Issue"]).toBe("ENG-142");
    expect(values["Title"]).toBe("Deploy pipeline fails on retry");
    expect(values["Labels"]).toBe("bug, factory");
    expect(values["Status"]).toBe("Todo");
    expect(values["Issue URL"]).toBe("https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry");
  });

  it("shows at most six values, with the timestamp first", () => {
    const values = onIssueLabelTriggerRenderer.getRootEventValues({ event: event(labelEvent) });

    expect(Object.keys(values).length).toBeLessThanOrEqual(6);
    expect(Object.keys(values)[0]).toBe("Received At");
  });

  it("falls back to dashes when the payload is empty", () => {
    const values = onIssueLabelTriggerRenderer.getRootEventValues({ event: event({}) });

    expect(values["Issue"]).toBe("-");
    expect(values["Labels"]).toBe("-");
    expect(values["Issue URL"]).toBe("-");
  });

  it("does not throw when there is no event", () => {
    expect(() => onIssueLabelTriggerRenderer.getRootEventValues({ event: undefined })).not.toThrow();
  });
});

describe("onIssueLabelTriggerRenderer.getTriggerProps", () => {
  it("renders the team and a label count", () => {
    const props = onIssueLabelTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { team: "t1", labels: ["l2", "l3"] },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "users", label: "Engineering" });
    expect(props.metadata?.[1]).toEqual({ icon: "tag", label: "2 labels" });
  });

  it("uses the singular form for a single label", () => {
    const props = onIssueLabelTriggerRenderer.getTriggerProps(
      buildTriggerContext({ node: { configuration: { team: "t1", labels: ["l2"] } } }),
    );

    expect(props.metadata?.[1]).toEqual({ icon: "tag", label: "1 label" });
  });

  it("surfaces the last event when one exists", () => {
    const props = onIssueLabelTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: { configuration: { team: "t1", labels: ["l2"] } },
        lastEvent: event(labelEvent),
      }),
    );

    expect(props.lastEventData?.title).toBe("ENG-142 · Deploy pipeline fails on retry");
    expect(props.lastEventData?.state).toBe("triggered");
  });

  it("does not throw when configuration and metadata are undefined", () => {
    expect(() =>
      onIssueLabelTriggerRenderer.getTriggerProps(
        buildTriggerContext({ node: { configuration: undefined, metadata: undefined } }),
      ),
    ).not.toThrow();
  });
});
