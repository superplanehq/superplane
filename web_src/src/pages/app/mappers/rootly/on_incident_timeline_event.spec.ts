import { afterEach, describe, expect, it } from "bun:test";
import { cleanup, render } from "@testing-library/react";
import { createElement } from "react";

import type { EventInfo, TriggerRendererContext } from "../types";
import { onEventTriggerRenderer } from "./on_incident_timeline_event";

function buildContext(configuration?: unknown, lastEvent?: EventInfo): TriggerRendererContext {
  return {
    node: {
      id: "node-1",
      name: "Incident timeline trigger",
      componentName: "rootly.onIncidentTimelineEvent",
      isCollapsed: false,
      configuration,
    },
    definition: {
      name: "rootly.onIncidentTimelineEvent",
      label: "On Incident Timeline Event",
      description: "",
      icon: "activity",
      color: "purple",
    },
    lastEvent,
  };
}

afterEach(cleanup);

describe("onEventTriggerRenderer.getTriggerProps", () => {
  it.each([
    { name: "missing configuration", configuration: undefined },
    { name: "empty configuration", configuration: {} },
    {
      name: "empty filters",
      configuration: { incidentStatus: [], severity: [], service: [], team: [], eventSource: [], visibility: "" },
    },
  ])("returns empty metadata for $name without a last event", ({ configuration }) => {
    const props = onEventTriggerRenderer.getTriggerProps(buildContext(configuration));

    expect(props.metadata).toEqual([]);
    expect(props.lastEventData).toBeUndefined();
  });

  it("preserves all six filters in display order with comma-space joining", () => {
    const props = onEventTriggerRenderer.getTriggerProps(
      buildContext({
        incidentStatus: ["started", "resolved"],
        severity: ["SEV1", "SEV2"],
        service: ["API", "Database"],
        team: ["Platform", "Support"],
        eventSource: ["slack", "web"],
        visibility: "private",
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "funnel", label: "Status: started, resolved" },
      { icon: "alert-circle", label: "Severity: SEV1, SEV2" },
      { icon: "layers", label: "Service: API, Database" },
      { icon: "users", label: "Team: Platform, Support" },
      { icon: "activity", label: "Source: slack, web" },
      { icon: "eye", label: "Visibility: private" },
    ]);
  });

  it.each([
    {
      name: "event text before incident title",
      data: {
        event: "Responder joined",
        kind: "timeline",
        incident: { title: "API outage", severity: "SEV1", status: "started" },
      },
      title: "Responder joined",
      content: "timeline · SEV1 · started",
    },
    {
      name: "incident title when event text is empty",
      data: { event: "", kind: "", incident: { title: "API outage", severity: "SEV2", status: "resolved" } },
      title: "API outage",
      content: "SEV2 · resolved",
    },
    {
      name: "default title with no subtitle content",
      data: { event: "", kind: "", incident: { title: "", severity: "", status: "" } },
      title: "Incident event",
      content: "",
    },
    {
      name: "kind and status with missing severity",
      data: { kind: "timeline", incident: { title: "API outage", status: "started" } },
      title: "API outage",
      content: "timeline · started",
    },
    {
      name: "missing event data",
      data: undefined,
      title: "Incident event",
      content: "",
    },
  ])("preserves last-event data for $name", ({ data, title, content }) => {
    const createdAt = new Date(Date.now() - 60_000).toISOString();
    const props = onEventTriggerRenderer.getTriggerProps(
      buildContext(undefined, {
        id: "event-1",
        nodeId: "node-1",
        type: "rootly.incident.timelineEvent",
        createdAt,
        data,
      }),
    );

    expect(props.metadata).toEqual([]);
    expect(props.lastEventData).toMatchObject({
      title,
      receivedAt: new Date(createdAt),
      state: "triggered",
      eventId: "event-1",
    });

    const { container } = render(createElement("div", null, props.lastEventData?.subtitle));
    const subtitle = container.textContent!;
    if (content) {
      expect(subtitle.startsWith(`${content} · `)).toBe(true);
      expect(subtitle.length).toBeGreaterThan(`${content} · `.length);
      return;
    }

    expect(subtitle.trim().length).toBeGreaterThan(0);
    expect(subtitle).not.toContain(" · ");
  });
});
