import { describe, expect, it } from "bun:test";
import { isValidElement } from "react";

import elasticIcon from "@/assets/icons/integrations/elastic.svg";
import { TimeAgo } from "@/components/TimeAgo";
import type { TriggerRendererContext } from "../types";
import { onCaseStatusChangeTriggerRenderer } from "./on_case_status_change";

function context(): TriggerRendererContext {
  return {
    node: {
      id: "node-1",
      name: "Case updates",
      componentName: "elastic.onCaseStatusChange",
      isCollapsed: false,
    },
    definition: {
      name: "elastic.onCaseStatusChange",
      label: "On Case Status Change",
      description: "",
      icon: "activity",
      color: "blue",
    },
    lastEvent: undefined,
  };
}

describe("onCaseStatusChangeTriggerRenderer.getTriggerProps", () => {
  it.each([undefined, {}, { cases: [], statuses: [], severities: [], tags: [] }])(
    "omits metadata for empty configuration %j",
    (configuration) => {
      const ctx = context();
      ctx.node.configuration = configuration;

      const props = onCaseStatusChangeTriggerRenderer.getTriggerProps(ctx);

      expect(props.metadata).toEqual([]);
      expect(props).not.toHaveProperty("lastEventData");
    },
  );

  it("keeps filter order and resolves case names with nullish fallback", () => {
    const ctx = context();
    ctx.node.configuration = {
      cases: ["case-1", "case-2", "case-3"],
      statuses: ["open", "closed"],
      severities: ["low", "high"],
      tags: [{ value: "security" }, {}, { value: "" }, { value: " urgent " }, { value: "security" }],
    };
    ctx.node.metadata = { caseNames: { "case-1": "Investigation", "case-3": "" } };

    expect(onCaseStatusChangeTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([
      { icon: "folder", label: "Investigation, case-2, " },
      { icon: "activity", label: "open, closed" },
      { icon: "alert-triangle", label: "low, high" },
      { icon: "tag", label: "security,  urgent , security" },
    ]);
  });

  it.each([undefined, {}, { caseNames: {} }])("uses case IDs when names are missing from %j", (metadata) => {
    const ctx = context();
    ctx.node.configuration = { cases: ["case-1", "case-2"] };
    ctx.node.metadata = metadata;

    expect(onCaseStatusChangeTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([
      { icon: "folder", label: "case-1, case-2" },
    ]);
  });

  it("keeps an empty-label tag item when a nonempty array has no values", () => {
    const ctx = context();
    ctx.node.configuration = { tags: [{}, { value: "" }] };

    expect(onCaseStatusChangeTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([{ icon: "tag", label: "" }]);
  });

  it.each([
    ["Case updates", "On Case Status Change", "Case updates"],
    ["", "On Case Status Change", "On Case Status Change"],
    ["", "", "Unnamed trigger"],
  ])("preserves trigger appearance with node name %j and label %j", (name, label, title) => {
    const ctx = context();
    ctx.node.name = name;
    ctx.definition.label = label;

    expect(onCaseStatusChangeTriggerRenderer.getTriggerProps(ctx)).toEqual({
      title,
      iconSrc: elasticIcon,
      collapsedBackground: "bg-blue-100",
      metadata: [],
    });
  });

  it.each([
    { data: { title: "Investigation", status: "closed" }, title: 'Case "Investigation" changed to closed' },
    { data: { title: "Investigation" }, title: 'Case "Investigation" changed' },
    { data: { status: "closed" }, title: "Case status changed" },
    { data: undefined, title: "Case status changed" },
  ])("preserves last-event props for $title", ({ data, title }) => {
    const ctx = context();
    const receivedAt = new Date(Date.now() - 60_000);
    ctx.node.configuration = { statuses: ["closed"] };
    ctx.lastEvent = {
      id: "event-1",
      createdAt: receivedAt.toISOString(),
      nodeId: ctx.node.id,
      type: "elastic.case.status.changed",
      data,
    };

    const props = onCaseStatusChangeTriggerRenderer.getTriggerProps(ctx);

    expect(props).toMatchObject({
      title: "Case updates",
      iconSrc: elasticIcon,
      collapsedBackground: "bg-blue-100",
      metadata: [{ icon: "activity", label: "closed" }],
      lastEventData: { title, receivedAt, state: "triggered", eventId: "event-1" },
    });
    const subtitle = props.lastEventData?.subtitle;
    expect(isValidElement(subtitle)).toBe(true);
    if (!isValidElement<{ date: Date }>(subtitle)) throw new Error("Expected a relative-time element");
    expect(subtitle.type).toBe(TimeAgo);
    expect(subtitle.props.date).toEqual(receivedAt);
  });
});
