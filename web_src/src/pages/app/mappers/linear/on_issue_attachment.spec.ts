import { describe, expect, it } from "vitest";
import type { ComponentDefinition, EventInfo, NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onIssueAttachmentTriggerRenderer } from "./on_issue_attachment";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-03-27T16:42:11Z").toISOString(),
    nodeId: "node-1",
    type: "linear.attachment",
    data,
  };
}

const attachmentEvent = {
  action: "create",
  type: "Attachment",
  actor: { id: "u1", name: "Ada Lovelace", email: "ada@example.com", type: "user" },
  data: {
    id: "a7c41e88",
    title: "Deploy - staging",
    subtitle: "Succeeded in 4m 12s",
    url: "https://app.superplane.com/runs/8f21c4d0",
    issueId: "2174add1",
    issue: {
      id: "2174add1",
      identifier: "ENG-142",
      title: "Deploy pipeline fails on retry",
      url: "https://linear.app/acme/issue/ENG-142",
    },
  },
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Issue Attachment",
    componentName: "linear.onIssueAttachment",
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
      name: "linear.onIssueAttachment",
      label: "On Issue Attachment",
      description: "",
      icon: "linear",
      color: "indigo",
      ...overrides?.definition,
    },
    lastEvent: overrides?.lastEvent,
  } as TriggerRendererContext;
}

describe("onIssueAttachmentTriggerRenderer.getTitleAndSubtitle", () => {
  it("builds the title from the attachment", () => {
    const context: TriggerEventContext = { event: event(attachmentEvent) };
    expect(onIssueAttachmentTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Deploy - staging");
  });

  it("falls back to a generic title when the payload is empty", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onIssueAttachmentTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Attachment");
  });

  it("does not throw when there is no event", () => {
    expect(() => onIssueAttachmentTriggerRenderer.getTitleAndSubtitle({ event: undefined })).not.toThrow();
  });
});

describe("onIssueAttachmentTriggerRenderer.getRootEventValues", () => {
  it("maps the attachment fields, including the enriched issue", () => {
    const values = onIssueAttachmentTriggerRenderer.getRootEventValues({ event: event(attachmentEvent) });

    expect(values["Received At"]).toBeDefined();
    expect(values["Issue"]).toBe("ENG-142 · Deploy pipeline fails on retry");
    expect(values["Title"]).toBe("Deploy - staging");
    expect(values["Subtitle"]).toBe("Succeeded in 4m 12s");
    expect(values["Action"]).toBe("Added");
    expect(values["Attachment URL"]).toBe("https://app.superplane.com/runs/8f21c4d0");
  });

  it("shows at most six values, with the timestamp first", () => {
    const values = onIssueAttachmentTriggerRenderer.getRootEventValues({ event: event(attachmentEvent) });

    expect(Object.keys(values).length).toBeLessThanOrEqual(6);
    expect(Object.keys(values)[0]).toBe("Received At");
  });

  it("humanises the remove action as Removed", () => {
    const values = onIssueAttachmentTriggerRenderer.getRootEventValues({
      event: event({ ...attachmentEvent, action: "remove" }),
    });

    expect(values["Action"]).toBe("Removed");
  });

  it("dashes the issue when enrichment did not run", () => {
    const values = onIssueAttachmentTriggerRenderer.getRootEventValues({
      event: event({ ...attachmentEvent, data: { id: "a1", title: "Deploy", issueId: "2174add1" } }),
    });

    expect(values["Issue"]).toBe("-");
    expect(values["Title"]).toBe("Deploy");
  });

  it("does not throw when there is no event", () => {
    expect(() => onIssueAttachmentTriggerRenderer.getRootEventValues({ event: undefined })).not.toThrow();
  });
});

describe("onIssueAttachmentTriggerRenderer.getTriggerProps", () => {
  it("renders the team and selected actions", () => {
    const props = onIssueAttachmentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { team: "t1", actions: ["create", "remove"] },
          metadata: { team: { id: "t1", key: "ENG", name: "Engineering" } },
        },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "users", label: "Engineering" });
    expect(props.metadata?.[1]).toEqual({ icon: "funnel", label: "Added, Removed" });
  });

  it("falls back to the configured team when metadata is missing", () => {
    const props = onIssueAttachmentTriggerRenderer.getTriggerProps(
      buildTriggerContext({ node: { configuration: { team: "t1", actions: ["create"] } } }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "users", label: "t1" });
  });

  it("surfaces the last event when one exists", () => {
    const props = onIssueAttachmentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: { configuration: { team: "t1", actions: ["create"] } },
        lastEvent: event(attachmentEvent),
      }),
    );

    expect(props.lastEventData?.title).toBe("Deploy - staging");
    expect(props.lastEventData?.state).toBe("triggered");
  });

  it("does not throw when configuration and metadata are undefined", () => {
    expect(() =>
      onIssueAttachmentTriggerRenderer.getTriggerProps(
        buildTriggerContext({ node: { configuration: undefined, metadata: undefined } }),
      ),
    ).not.toThrow();
  });
});
