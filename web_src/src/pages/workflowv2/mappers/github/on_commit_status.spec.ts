import { describe, expect, it } from "vitest";

import type { EventInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onCommitStatusTriggerRenderer } from "./on_commit_status";

const definition = {
  name: "github.onCommitStatus",
  label: "On Commit Status",
  description: "",
  icon: "github",
  color: "gray",
};

describe("onCommitStatusTriggerRenderer", () => {
  it("builds title from status context, state, and SHA", () => {
    const event = statusEvent({
      state: "success",
      context: "ci/build",
      sha: "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
    });

    const context: TriggerEventContext = { event };

    expect(onCommitStatusTriggerRenderer.getTitleAndSubtitle(context).title).toBe("ci/build succeeded - d6f3c8a");
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
      commit: {
        author: { login: "monalisa" },
        commit: {
          message: "Deploy checkout validation\n\nSigned-off-by: Mona Lisa <mona@example.com>",
        },
        html_url: "https://github.com/acme/snaketoy/commit/d6f3c8a",
      },
    });

    const values = onCommitStatusTriggerRenderer.getRootEventValues({ event });

    expect(Object.keys(values)).toEqual([
      "State",
      "Context",
      "Description",
      "SHA",
      "Commit message",
      "Commit author",
      "Branches",
      "Repository",
      "Status creator",
      "Commit URL",
      "Target URL",
    ]);
    expect(values.State).toBe("failure");
    expect(values.Context).toBe("deploy/production");
    expect(values.Description).toBe("Deployment failed");
    expect(values.SHA).toBe("d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44");
    expect(values["Commit message"]).toBe("Deploy checkout validation");
    expect(values["Commit author"]).toBe("monalisa");
    expect(values.Branches).toBe("main, release/v1");
    expect(values.Repository).toBe("acme/snaketoy");
    expect(values["Status creator"]).toBe("octocat");
  });

  it("renders repository and filters as node metadata", () => {
    const context: TriggerRendererContext = {
      node: {
        id: "node-1",
        name: "Status trigger",
        componentName: "github.onCommitStatus",
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

    const props = onCommitStatusTriggerRenderer.getTriggerProps(context);

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
