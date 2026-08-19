import { describe, expect, it } from "vitest";
import type { ComponentDefinition, EventInfo, NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onIssueCommentTriggerRenderer } from "./on_issue_comment";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-07-17T09:30:00Z").toISOString(),
    nodeId: "node-1",
    type: "jira.issueComment",
    data,
  };
}

const commentData = {
  action: "created",
  comment: {
    id: "10019",
    body: "Thanks for the update, looking into this now.",
    author: { displayName: "Alice Smith" },
  },
  issue: {
    id: "10001",
    key: "ENG-42",
    self: "https://your-domain.atlassian.net/rest/api/3/issue/10001",
    fields: { summary: "Login page returns 500 on invalid password" },
  },
};

describe("onIssueCommentTriggerRenderer", () => {
  it("derives an issue title from the event", () => {
    const context: TriggerEventContext = { event: event(commentData) };
    expect(onIssueCommentTriggerRenderer.getTitleAndSubtitle(context).title).toBe(
      "ENG-42 - Login page returns 500 on invalid password",
    );
  });

  it("falls back to a generic title when no issue is present", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onIssueCommentTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Comment event");
  });

  it("maps the comment fields to root event values", () => {
    const context: TriggerEventContext = { event: event(commentData) };
    const values = onIssueCommentTriggerRenderer.getRootEventValues(context);
    expect(values["Received At"]).toBeDefined();
    expect(values["Action"]).toBe("Created");
    expect(values["Key"]).toBe("ENG-42");
    expect(values["Comment"]).toBe("Thanks for the update, looking into this now.");
    expect(values["Author"]).toBe("Alice Smith");
  });

  it("extracts plain text from an Atlassian Document Format comment body", () => {
    const adfBody = {
      type: "doc",
      content: [
        {
          type: "paragraph",
          content: [{ type: "text", text: "Looking into this now." }],
        },
      ],
    };
    const context: TriggerEventContext = {
      event: event({ ...commentData, comment: { ...commentData.comment, body: adfBody } }),
    };
    expect(onIssueCommentTriggerRenderer.getRootEventValues(context)["Comment"]).toBe("Looking into this now.");
  });

  it("includes a mention's attrs.text when extracting plain text", () => {
    const adfBody = {
      type: "doc",
      content: [
        {
          type: "paragraph",
          content: [
            { type: "mention", attrs: { id: "abc", text: "@Alice Smith" } },
            { type: "text", text: " please take a look" },
          ],
        },
      ],
    };
    const context: TriggerEventContext = {
      event: event({ ...commentData, comment: { ...commentData.comment, body: adfBody } }),
    };
    expect(onIssueCommentTriggerRenderer.getRootEventValues(context)["Comment"]).toBe(
      "@Alice Smith please take a look",
    );
  });

  it("separates text across paragraphs with a space", () => {
    const adfBody = {
      type: "doc",
      content: [
        { type: "paragraph", content: [{ type: "text", text: "Hello" }] },
        { type: "paragraph", content: [{ type: "text", text: "World" }] },
      ],
    };
    const context: TriggerEventContext = {
      event: event({ ...commentData, comment: { ...commentData.comment, body: adfBody } }),
    };
    expect(onIssueCommentTriggerRenderer.getRootEventValues(context)["Comment"]).toBe("Hello World");
  });

  it("skips empty paragraphs when joining block text", () => {
    const adfBody = {
      type: "doc",
      content: [
        { type: "paragraph", content: [{ type: "text", text: "Hello" }] },
        { type: "paragraph" },
        { type: "paragraph", content: [{ type: "text", text: "World" }] },
      ],
    };
    const context: TriggerEventContext = {
      event: event({ ...commentData, comment: { ...commentData.comment, body: adfBody } }),
    };
    expect(onIssueCommentTriggerRenderer.getRootEventValues(context)["Comment"]).toBe("Hello World");
  });

  it("falls back to a dash when the comment body is missing", () => {
    const context: TriggerEventContext = {
      event: event({ ...commentData, comment: { ...commentData.comment, body: undefined } }),
    };
    expect(onIssueCommentTriggerRenderer.getRootEventValues(context)["Comment"]).toBe("-");
  });
});

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Issue Comment",
    componentName: "jira.onIssueComment",
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
      name: "jira.onIssueComment",
      label: "On Issue Comment",
      description: "",
      icon: "jira",
      color: "blue",
      ...overrides?.definition,
    },
    lastEvent: overrides?.lastEvent,
  } as TriggerRendererContext;
}

describe("onIssueCommentTriggerRenderer.getTriggerProps", () => {
  it("renders the project and selected events", () => {
    const props = onIssueCommentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { project: "ENG", events: ["created", "updated"] },
          metadata: { project: { key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "folder", label: "Engineering (ENG)" });
    expect(props.metadata?.[1]).toEqual({ icon: "funnel", label: "Created, Updated" });
  });

  it("renders the content filter after the selected events", () => {
    const props = onIssueCommentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { project: "ENG", events: ["created"], contentFilter: "urgent" },
          metadata: { project: { key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata?.[1]).toEqual({ icon: "funnel", label: "Created" });
    expect(props.metadata?.[2]).toEqual({ icon: "funnel", label: "Filter: urgent" });
  });

  it("omits the content filter when it is not configured", () => {
    const props = onIssueCommentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: { configuration: { project: "ENG", events: ["created"] } },
      }),
    );

    expect(props.metadata).toHaveLength(2);
  });
});
