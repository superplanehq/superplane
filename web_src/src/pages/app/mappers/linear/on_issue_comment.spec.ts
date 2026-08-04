import { describe, expect, it } from "vitest";
import type { ComponentDefinition, EventInfo, NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onIssueCommentTriggerRenderer } from "./on_issue_comment";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-03-27T16:42:11Z").toISOString(),
    nodeId: "node-1",
    type: "linear.comment",
    data,
  };
}

const commentEvent = {
  action: "create",
  type: "Comment",
  url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry#comment-c1",
  actor: { id: "u1", name: "Ada Lovelace", email: "ada@example.com", type: "user" },
  data: {
    id: "c1",
    body: "Deploy pipeline is green again.",
    url: "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry#comment-c1",
    user: { id: "u1", name: "Ada Lovelace", displayName: "ada", email: "ada@example.com" },
    issue: { id: "i1", identifier: "ENG-142", title: "Deploy pipeline fails on retry" },
  },
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Issue Comment",
    componentName: "linear.onIssueComment",
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
      name: "linear.onIssueComment",
      label: "On Issue Comment",
      description: "",
      icon: "linear",
      color: "indigo",
      ...overrides?.definition,
    },
    lastEvent: overrides?.lastEvent,
  } as TriggerRendererContext;
}

describe("onIssueCommentTriggerRenderer.getTitleAndSubtitle", () => {
  it("builds the title from the commented issue", () => {
    const context: TriggerEventContext = { event: event(commentEvent) };
    expect(onIssueCommentTriggerRenderer.getTitleAndSubtitle(context).title).toBe(
      "ENG-142 · Deploy pipeline fails on retry",
    );
  });

  it("falls back to a generic title when the payload is empty", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onIssueCommentTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Comment");
  });

  it("does not throw when there is no event", () => {
    expect(() => onIssueCommentTriggerRenderer.getTitleAndSubtitle({ event: undefined })).not.toThrow();
  });
});

describe("onIssueCommentTriggerRenderer.getRootEventValues", () => {
  it("maps the comment fields, including the author and a link to the comment", () => {
    const values = onIssueCommentTriggerRenderer.getRootEventValues({ event: event(commentEvent) });

    expect(values["Received At"]).toBeDefined();
    expect(values["Issue"]).toBe("ENG-142 · Deploy pipeline fails on retry");
    expect(values["Author"]).toBe("ada");
    expect(values["Comment"]).toBe("Deploy pipeline is green again.");
    expect(values["Action"]).toBe("Created");
    expect(values["Comment URL"]).toBe(
      "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails-on-retry#comment-c1",
    );
  });

  it("shows at most six values, with the timestamp first", () => {
    const values = onIssueCommentTriggerRenderer.getRootEventValues({ event: event(commentEvent) });

    expect(Object.keys(values).length).toBeLessThanOrEqual(6);
    expect(Object.keys(values)[0]).toBe("Received At");
  });

  it("humanises the remove action as Deleted", () => {
    const values = onIssueCommentTriggerRenderer.getRootEventValues({
      event: event({ ...commentEvent, action: "remove" }),
    });

    expect(values["Action"]).toBe("Deleted");
  });

  it("falls back to dashes when the payload is empty", () => {
    const values = onIssueCommentTriggerRenderer.getRootEventValues({ event: event({}) });

    expect(values["Issue"]).toBe("-");
    expect(values["Author"]).toBe("-");
    expect(values["Comment URL"]).toBe("-");
  });

  it("does not throw when there is no event", () => {
    expect(() => onIssueCommentTriggerRenderer.getRootEventValues({ event: undefined })).not.toThrow();
  });
});

describe("onIssueCommentTriggerRenderer.getTriggerProps", () => {
  it("renders the team and selected actions", () => {
    const props = onIssueCommentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { team: "t1", actions: ["create", "update"] },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "users", label: "Engineering" });
    expect(props.metadata?.[1]).toEqual({ icon: "funnel", label: "Created, Updated" });
  });

  it("falls back to the configured team when metadata is missing", () => {
    const props = onIssueCommentTriggerRenderer.getTriggerProps(
      buildTriggerContext({ node: { configuration: { team: "t1", actions: ["create"] } } }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "users", label: "t1" });
  });

  it("renders the content filter after the selected actions", () => {
    const props = onIssueCommentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { team: "t1", actions: ["create"], contentFilter: "^/deploy" },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata?.[1]).toEqual({ icon: "funnel", label: "Created" });
    expect(props.metadata?.[2]).toEqual({ icon: "funnel", label: "Filter: ^/deploy" });
  });

  it("omits the content filter when it is not configured", () => {
    const props = onIssueCommentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { team: "t1", actions: ["create"] },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata).toHaveLength(2);
  });

  it("surfaces the last event when one exists", () => {
    const props = onIssueCommentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: { configuration: { team: "t1", actions: ["create"] } },
        lastEvent: event(commentEvent),
      }),
    );

    expect(props.lastEventData?.title).toBe("ENG-142 · Deploy pipeline fails on retry");
    expect(props.lastEventData?.state).toBe("triggered");
  });

  it("does not throw when configuration and metadata are undefined", () => {
    expect(() =>
      onIssueCommentTriggerRenderer.getTriggerProps(
        buildTriggerContext({ node: { configuration: undefined, metadata: undefined } }),
      ),
    ).not.toThrow();
  });
});
