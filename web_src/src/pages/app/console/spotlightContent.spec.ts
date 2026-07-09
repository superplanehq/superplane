import { describe, expect, it } from "vitest";

import {
  applySourceDefaults,
  checksAreStages,
  DEFAULT_SPOTLIGHT_CONTENT,
  normalizeStatus,
  slotDefaultsForSource,
  spotlightContentToYaml,
  spotlightPropsFromContent,
  validateSpotlightContent,
  type SpotlightPanelContent,
} from "./spotlightContent";

/** A row shaped like the `runs` data source: derived fields + raw `executions`. */
const runRow = {
  status: "passed",
  nodeName: "Deploy Helm — Production",
  createdAt: "2026-07-01T10:00:00Z",
  durationMs: 26000,
  executions: [
    { nodeName: "SaaS CI", state: "STATE_FINISHED", result: "RESULT_PASSED" },
    { nodeName: "Analyze", state: "STATE_FINISHED", result: "RESULT_FAILED" },
    { nodeName: "Promote", state: "STATE_STARTED", result: "RESULT_NONE" },
  ],
};

/** A memory-shaped record with an explicit checks payload and person slots. */
const deploymentRow = {
  status: "passed",
  status_label: "Live",
  repo: "acme/superplane · main",
  deployed_at: "2026-07-01T10:00:00Z",
  duration_ms: 252000,
  deployed_by: { name: "Ada", avatar_url: "https://example.com/ada.png" },
  approved_by: { name: "Grace", avatar_url: "https://example.com/grace.png" },
  pr: { title: "#42 Ship it", url: "https://example.com/pr/42" },
  checks: [
    { name: "build", status: "passed" },
    { name: "e2e", status: "failed" },
  ],
};

const memoryContent: SpotlightPanelContent = {
  ...DEFAULT_SPOTLIGHT_CONTENT,
  dataSource: { kind: "memory", namespace: "deployments" },
  statusLabelField: "status_label",
  actorNameField: "deployed_by.name",
  actorAvatarField: "deployed_by.avatar_url",
  titleField: "pr.title",
  hrefField: "pr.url",
  subtitleField: "repo",
  timestampField: "deployed_at",
  durationField: "duration_ms",
  approverNameField: "approved_by.name",
  approverAvatarField: "approved_by.avatar_url",
  checksField: "checks",
  checkNameField: "name",
  checkStatusField: "status",
};

describe("spotlightPropsFromContent — runs-primary default", () => {
  it("resolves the default run slots off a run row", () => {
    const props = spotlightPropsFromContent(DEFAULT_SPOTLIGHT_CONTENT, runRow);
    expect(props.kicker).toBe("Latest run");
    expect(props.status).toBe("success");
    expect(props.statusLabel).toBe("passed");
    expect(props.title).toBe("Deploy Helm — Production");
    expect(props.timestamp).toBe("2026-07-01T10:00:00Z");
    expect(props.duration).toBe(26000);
    // No person slots by default for a run.
    expect(props.actor).toBeUndefined();
    expect(props.approver).toBeUndefined();
  });

  it("reads run stages from the executions array, mapping RESULT_/STATE_ tokens", () => {
    const props = spotlightPropsFromContent(DEFAULT_SPOTLIGHT_CONTENT, runRow);
    expect(props.checks).toEqual([
      { name: "SaaS CI", status: "success" },
      { name: "Analyze", status: "failed" },
      // RESULT_NONE is inconclusive, so the running STATE_STARTED wins.
      { name: "Promote", status: "running" },
    ]);
  });
});

describe("spotlightPropsFromContent — memory mapping", () => {
  it("resolves nested dot-path slots off a single record", () => {
    const props = spotlightPropsFromContent(memoryContent, deploymentRow);
    expect(props.status).toBe("success");
    expect(props.statusLabel).toBe("Live");
    expect(props.actor).toEqual({ name: "Ada", avatarUrl: "https://example.com/ada.png" });
    expect(props.title).toBe("#42 Ship it");
    expect(props.href).toBe("https://example.com/pr/42");
    expect(props.subtitle).toBe("acme/superplane · main");
    expect(props.timestamp).toBe("2026-07-01T10:00:00Z");
    expect(props.duration).toBe(252000);
    expect(props.approver).toEqual({ name: "Grace", avatarUrl: "https://example.com/grace.png" });
  });

  it("normalizes the overall status synonyms to a known status", () => {
    const props = spotlightPropsFromContent(memoryContent, { ...deploymentRow, status: "in_progress" });
    expect(props.status).toBe("running");
  });

  it("resolves the checks array and normalizes each item status", () => {
    const props = spotlightPropsFromContent(memoryContent, deploymentRow);
    expect(props.checks).toEqual([
      { name: "build", status: "success" },
      { name: "e2e", status: "failed" },
    ]);
  });

  it("supports custom check item sub-paths", () => {
    const content: SpotlightPanelContent = {
      ...memoryContent,
      checksField: "gates",
      checkNameField: "label",
      checkStatusField: "result.state",
    };
    const props = spotlightPropsFromContent(content, {
      ...deploymentRow,
      gates: [{ label: "security", result: { state: "warn" } }],
    });
    expect(props.checks).toEqual([{ name: "security", status: "warning" }]);
  });

  it("returns undefined checks when the field is missing or not an array", () => {
    expect(spotlightPropsFromContent(memoryContent, { pr: { title: "x" } }).checks).toBeUndefined();
    const scalarChecks = spotlightPropsFromContent(memoryContent, { ...deploymentRow, checks: "nope" });
    expect(scalarChecks.checks).toBeUndefined();
  });

  it("omits a person entirely when neither name nor avatar resolve", () => {
    const props = spotlightPropsFromContent(memoryContent, { pr: { title: "x" } });
    expect(props.actor).toBeUndefined();
    expect(props.approver).toBeUndefined();
  });
});

describe("normalizeStatus", () => {
  it("maps SuperPlane run/execution enum tokens", () => {
    expect(normalizeStatus("RESULT_PASSED")).toBe("success");
    expect(normalizeStatus("RESULT_FAILED")).toBe("failed");
    expect(normalizeStatus("STATE_STARTED")).toBe("running");
    expect(normalizeStatus("RESULT_NONE")).toBe("neutral");
  });

  it("falls back to neutral for unknown values", () => {
    expect(normalizeStatus("whatever")).toBe("neutral");
    expect(normalizeStatus(undefined)).toBe("neutral");
  });
});

describe("source-aware defaults", () => {
  it("marks only runs as stage-backed", () => {
    expect(checksAreStages("runs")).toBe(true);
    expect(checksAreStages("memory")).toBe(false);
    expect(checksAreStages("executions")).toBe(false);
  });

  it("points runs checks at the executions array and executions at no array", () => {
    expect(slotDefaultsForSource("runs").checksField).toBe("executions");
    expect(slotDefaultsForSource("executions").checksField).toBe("");
    expect(slotDefaultsForSource("memory").checksField).toBe("checks");
  });

  it("swaps slot defaults on source change while keeping the kicker", () => {
    const next = applySourceDefaults(
      { ...DEFAULT_SPOTLIGHT_CONTENT, kicker: "Keep me" },
      {
        kind: "memory",
        namespace: "deployments",
      },
    );
    expect(next.kicker).toBe("Keep me");
    expect(next.dataSource).toEqual({ kind: "memory", namespace: "deployments" });
    expect(next.checksField).toBe("checks");
    expect(next.titleField).toBe("");
  });
});

describe("validateSpotlightContent", () => {
  it("accepts the default content", () => {
    expect(validateSpotlightContent(DEFAULT_SPOTLIGHT_CONTENT)).toBeNull();
  });

  it("requires a memory namespace", () => {
    const content: SpotlightPanelContent = {
      ...DEFAULT_SPOTLIGHT_CONTENT,
      dataSource: { kind: "memory", namespace: "  " },
    };
    expect(validateSpotlightContent(content)).toMatch(/namespace/i);
  });

  it("requires a title or actor headline", () => {
    const content: SpotlightPanelContent = {
      ...DEFAULT_SPOTLIGHT_CONTENT,
      titleField: "",
      actorNameField: "",
    };
    expect(validateSpotlightContent(content)).toMatch(/headline/i);
  });
});

describe("spotlightContentToYaml", () => {
  it("serializes to the type + dataSource + render shape", () => {
    const yamlText = spotlightContentToYaml(DEFAULT_SPOTLIGHT_CONTENT);
    expect(yamlText).toContain("type: spotlight");
    expect(yamlText).toContain("kind: runs");
    expect(yamlText).toContain("title: nodeName");
    expect(yamlText).toContain("field: executions");
  });

  it("omits empty slot groups", () => {
    const content: SpotlightPanelContent = {
      ...DEFAULT_SPOTLIGHT_CONTENT,
      approverNameField: "",
      approverAvatarField: "",
      approverLabel: "",
      checksField: "",
    };
    const yamlText = spotlightContentToYaml(content);
    expect(yamlText).not.toContain("approver:");
    expect(yamlText).not.toContain("checks:");
  });
});
