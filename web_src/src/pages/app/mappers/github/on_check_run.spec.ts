import { describe, expect, it } from "vitest";

import type { EventInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onCheckRunTriggerRenderer } from "./on_check_run";

const definition = {
  name: "github.onCheckRun",
  label: "On Check Run",
  description: "",
  icon: "github",
  color: "gray",
};

describe("onCheckRunTriggerRenderer", () => {
  it("builds title from check run name, conclusion, and SHA", () => {
    const event = checkRunEvent({
      check_run: {
        name: "DCO",
        status: "completed",
        conclusion: "success",
        head_sha: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
      },
    });

    const context: TriggerEventContext = { event };

    expect(onCheckRunTriggerRenderer.getTitleAndSubtitle(context).title).toBe("DCO succeeded - d6f3c8a");
  });

  it("returns curated check run details for the sidebar", () => {
    const event = checkRunEvent({
      action: "completed",
      check_run: {
        name: "Cloudflare Pages",
        status: "completed",
        conclusion: "failure",
        head_sha: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
        details_url: "https://dash.cloudflare.com/example",
        app: { name: "Cloudflare Pages" },
        check_suite: { head_branch: "feature/checks" },
        pull_requests: [{ number: 42 }],
      },
      repository: { full_name: "acme/snaketoy" },
    });

    const values = onCheckRunTriggerRenderer.getRootEventValues({ event });

    expect(Object.keys(values)).toEqual([
      "Action",
      "Name",
      "Status",
      "Conclusion",
      "Branch",
      "SHA",
      "Pull request",
      "App",
      "Details URL",
      "Repository",
    ]);
    expect(values.Name).toBe("Cloudflare Pages");
    expect(values.Conclusion).toBe("failure");
    expect(values.Branch).toBe("feature/checks");
    expect(values["Pull request"]).toBe("#42");
    expect(values.App).toBe("Cloudflare Pages");
  });

  it("renders repository and filters as node metadata", () => {
    const context: TriggerRendererContext = {
      node: {
        id: "node-1",
        name: "Check trigger",
        componentName: "github.onCheckRun",
        isCollapsed: false,
        metadata: {
          repository: {
            id: "repo-1",
            name: "acme/snaketoy",
            url: "https://github.com/acme/snaketoy",
          },
        },
        configuration: {
          statuses: ["completed"],
          conclusions: ["success"],
          names: [{ type: "equals", value: "DCO" }],
          branches: [{ type: "matches", value: "feature/.*" }],
          pullRequestsOnly: true,
        },
      },
      definition,
      lastEvent: undefined,
    };

    const props = onCheckRunTriggerRenderer.getTriggerProps(context);

    expect(props.metadata?.map((item) => item.label)).toEqual([
      "acme/snaketoy",
      "completed",
      "conclusion success",
      "name =DCO",
      "branch ~feature/.*",
      "pull requests only",
    ]);
  });
});

function checkRunEvent(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date().toISOString(),
    nodeId: "node-1",
    type: "github.checkRun",
    data,
  };
}
