import { describe, expect, it } from "vitest";

import type { EventInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onStatusTriggerRenderer } from "./on_status";

const definition = {
  name: "github.onStatus",
  label: "On Status",
  description: "",
  icon: "github",
  color: "gray",
};

describe("onStatusTriggerRenderer", () => {
  it("builds title from status context and SHA", () => {
    const event = statusEvent({
      state: "success",
      context: "ci/build",
      sha: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
    });

    const context: TriggerEventContext = { event };

    expect(onStatusTriggerRenderer.getTitleAndSubtitle(context).title).toBe("ci/build - d6f3c8a");
  });

  it("returns status details for the sidebar", () => {
    const event = statusEvent({
      state: "failure",
      context: "deploy/production",
      sha: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
      description: "Deployment failed",
      target_url: "https://example.com/build/1",
      branches: [{ name: "main" }, { name: "release/v1" }],
      repository: { full_name: "acme/snaketoy" },
      sender: { login: "octocat" },
      commit: { html_url: "https://github.com/acme/snaketoy/commit/d6f3c8a" },
    });

    const values = onStatusTriggerRenderer.getRootEventValues({ event });

    expect(values.State).toBe("failure");
    expect(values.Context).toBe("deploy/production");
    expect(values.Branches).toBe("main, release/v1");
    expect(values.Repository).toBe("acme/snaketoy");
    expect(values.Sender).toBe("octocat");
  });

  it("renders repository and filters as node metadata", () => {
    const context: TriggerRendererContext = {
      node: {
        id: "node-1",
        name: "Status trigger",
        componentName: "github.onStatus",
        isCollapsed: false,
        metadata: {
          repository: {
            id: "repo-1",
            name: "acme/snaketoy",
            url: "https://github.com/acme/snaketoy",
          },
        },
        configuration: {
          states: ["success", "failure"],
          contexts: [{ type: "matches", value: "ci/.*" }],
          branches: [{ type: "equals", value: "main" }],
        },
      },
      definition,
      lastEvent: undefined,
    };

    const props = onStatusTriggerRenderer.getTriggerProps(context);

    expect(props.metadata?.map((item) => item.label)).toEqual([
      "acme/snaketoy",
      "success, failure",
      "context ~ci/.*",
      "branch =main",
    ]);
  });
});

function statusEvent(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date().toISOString(),
    nodeId: "node-1",
    type: "github.status",
    data,
  };
}
