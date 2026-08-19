import { describe, expect, it } from "vitest";
import type { ComponentDefinition, EventInfo, NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onIssueTriggerRenderer } from "./on_issue";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-03-26T19:29:35Z").toISOString(),
    nodeId: "node-1",
    type: "linear.issue",
    data,
  };
}

const issueEvent = {
  action: "create",
  type: "Issue",
  url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry",
  actor: { id: "u1", name: "John Doe", email: "john@example.com", type: "user" },
  data: {
    id: "2174add1",
    identifier: "ENG-142",
    title: "Deploy pipeline fails on retry",
    state: { id: "s1", name: "Todo", type: "unstarted" },
    team: { id: "t1", key: "ENG", name: "Engineering" },
    labels: [{ id: "l1", name: "bug" }],
  },
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Issue",
    componentName: "linear.onIssue",
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
      name: "linear.onIssue",
      label: "On Issue",
      description: "",
      icon: "linear",
      color: "indigo",
      ...overrides?.definition,
    },
    lastEvent: overrides?.lastEvent,
  } as TriggerRendererContext;
}

describe("onIssueTriggerRenderer.getTitleAndSubtitle", () => {
  it("builds the title from the issue identifier and title", () => {
    const context: TriggerEventContext = { event: event(issueEvent) };
    expect(onIssueTriggerRenderer.getTitleAndSubtitle(context).title).toBe("ENG-142 · Deploy pipeline fails on retry");
  });

  it("falls back to a generic title when the payload is empty", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onIssueTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Issue");
  });

  it("uses whichever half of the label exists", () => {
    const context: TriggerEventContext = { event: event({ data: { title: "Only a title" } }) };
    expect(onIssueTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Only a title");
  });

  it("does not throw when there is no event", () => {
    expect(() => onIssueTriggerRenderer.getTitleAndSubtitle({ event: undefined })).not.toThrow();
  });
});

describe("onIssueTriggerRenderer.getRootEventValues", () => {
  it("maps the issue fields, including a link to the issue", () => {
    const values = onIssueTriggerRenderer.getRootEventValues({ event: event(issueEvent) });

    expect(values["Received At"]).toBeDefined();
    expect(values["Issue"]).toBe("ENG-142");
    expect(values["Title"]).toBe("Deploy pipeline fails on retry");
    expect(values["Action"]).toBe("Created");
    expect(values["Status"]).toBe("Todo");
    expect(values["Issue URL"]).toBe("https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry");
  });

  it("shows at most six values, with the timestamp first", () => {
    const values = onIssueTriggerRenderer.getRootEventValues({ event: event(issueEvent) });

    expect(Object.keys(values).length).toBeLessThanOrEqual(6);
    expect(Object.keys(values)[0]).toBe("Received At");
  });

  it("humanises the remove action as Deleted", () => {
    const values = onIssueTriggerRenderer.getRootEventValues({
      event: event({ ...issueEvent, action: "remove" }),
    });

    expect(values["Action"]).toBe("Deleted");
  });

  it("passes an unknown action through unchanged", () => {
    const values = onIssueTriggerRenderer.getRootEventValues({
      event: event({ ...issueEvent, action: "archived" }),
    });

    expect(values["Action"]).toBe("archived");
  });

  it("falls back to dashes when the payload is empty", () => {
    const values = onIssueTriggerRenderer.getRootEventValues({ event: event({}) });

    expect(values["Issue"]).toBe("-");
    expect(values["Title"]).toBe("-");
    expect(values["Issue URL"]).toBe("-");
  });

  it("does not throw when there is no event", () => {
    expect(() => onIssueTriggerRenderer.getRootEventValues({ event: undefined })).not.toThrow();
  });
});

describe("onIssueTriggerRenderer.getTriggerProps", () => {
  it("renders the team, selected actions and label filters", () => {
    const props = onIssueTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: {
            team: "t1",
            actions: ["create", "update"],
            labels: [{ type: "equals", value: "backend" }],
          },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "users", label: "Engineering" });
    expect(props.metadata?.[1]).toEqual({ icon: "funnel", label: "Created, Updated" });
    expect(props.metadata?.[2]?.icon).toBe("tag");
  });

  it("falls back to the configured team when metadata is missing", () => {
    const props = onIssueTriggerRenderer.getTriggerProps(
      buildTriggerContext({ node: { configuration: { team: "t1", actions: ["create"] } } }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "users", label: "t1" });
  });

  it("omits the label badge when no label filter is configured", () => {
    const props = onIssueTriggerRenderer.getTriggerProps(
      buildTriggerContext({ node: { configuration: { team: "t1", actions: ["create"] } } }),
    );

    expect((props.metadata || []).map((item) => item.icon)).not.toContain("tag");
  });

  it("surfaces the last event when one exists", () => {
    const props = onIssueTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: { configuration: { team: "t1", actions: ["create"] } },
        lastEvent: event(issueEvent),
      }),
    );

    expect(props.lastEventData?.title).toBe("ENG-142 · Deploy pipeline fails on retry");
    expect(props.lastEventData?.state).toBe("triggered");
  });

  it("does not throw when configuration and metadata are undefined", () => {
    expect(() =>
      onIssueTriggerRenderer.getTriggerProps(
        buildTriggerContext({ node: { configuration: undefined, metadata: undefined } }),
      ),
    ).not.toThrow();
  });
});
