import { describe, expect, it } from "bun:test";

import type { EventInfo } from "../types";
import type { ArtifactPushData, OccurrenceData } from "./artifact_registry";
import { getArtifactAnalysisMapper, getArtifactMapper } from "./artifact_registry_mapper";
import { onArtifactAnalysisTriggerRenderer } from "./on_artifact_analysis";
import { onArtifactPushTriggerRenderer } from "./on_artifact_push";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

function artifactEvent(type: string, data?: ArtifactPushData | OccurrenceData): EventInfo {
  return { id: "evt-1", createdAt: new Date().toISOString(), nodeId: "node-1", type, data };
}

describe("artifact trigger titles", () => {
  it.each([undefined, artifactEvent("gcp.onArtifactPush")])("uses generic titles without event data (%j)", (event) => {
    expect(onArtifactPushTriggerRenderer.getTitleAndSubtitle({ event }).title).toBe("Pushed artifact image");
    expect(onArtifactAnalysisTriggerRenderer.getTitleAndSubtitle({ event }).title).toBe("Analysis event");
  });

  it.each<[string | undefined, string, string]>([
    [undefined, "Pushed artifact image", "Analysis event"],
    ["", "Pushed artifact image", "Analysis event"],
    ["  \t ", "Pushed artifact image", "Analysis event"],
    ["invalid", "Pushed artifact image", "Analysis event"],
    ["host/", "Pushed artifact image", "Analysis event"],
    ["host/project", "Pushed project", "Analysis: project"],
    ["host/project/repository", "Pushed project/repository", "Analysis: project/repository"],
    ["host/project/repository/image:v1", "Pushed image:v1", "Analysis: image:v1"],
    ["http://host/project/repository/image:v1", "Pushed image:v1", "Analysis: image:v1"],
    ["https://host/project/repository/image:v1", "Pushed image:v1", "Analysis: image:v1"],
    ["  https://host/project/repository/image:v1 \t", "Pushed image:v1", "Analysis: image:v1"],
    ["host/project/repository/team/image:v1", "Pushed image:v1", "Analysis: image:v1"],
    ["host/project/repository/team/image@sha256:abcdef123456", "Pushed image@abcdef12", "Analysis: image@abcdef12"],
    ["host/project/repository/image@sha256:abc", "Pushed image@abc", "Analysis: image@abc"],
    ["host/project/repository/image:abcdefghijkl", "Pushed image:abcdefghijkl", "Analysis: image:abcdefghijkl"],
    ["host/project/repository/image:abcdefghijklm", "Pushed image:abcdefghijkl...", "Analysis: image:abcdefghijkl..."],
    ["host/project/repository/image@abcdefghijkl", "Pushed image@abcdefghijkl", "Analysis: image@abcdefghijkl"],
    [
      "host/project/repository/image@sha512:abcdef123456",
      "Pushed image@sha512:abcde...",
      "Analysis: image@sha512:abcde...",
    ],
    ["host/project/repository/team/image", "Pushed image", "Analysis: image"],
    ["host/project/repository/team:parent/image", "Pushed image", "Analysis: image"],
    ["host/project/repository/image:", "Pushed image", "Analysis: image"],
    ["host/project/repository/image@", "Pushed image", "Analysis: image"],
    ["host/project/repository/image@sha256:", "Pushed image@", "Analysis: image@"],
    ["host/project/repository/team/", "Pushed team/", "Analysis: team/"],
    ["host/project/repository/", "Pushed project/repository/", "Analysis: project/repository/"],
    ["host/project/repository/@", "Pushed project/repository/@", "Analysis: project/repository/@"],
    ["host/project/repository/@abc", "Pushed @abc", "Analysis: @abc"],
    ["host/project/repository/image:v1@sha256:abcdef123456", "Pushed image:v1@abcdef12", "Analysis: image:v1@abcdef12"],
  ])("formats %j", (reference, pushTitle, analysisTitle) => {
    const pushData: ArtifactPushData = { digest: reference };
    const analysisData: OccurrenceData = { resourceUri: reference };
    expect(
      onArtifactPushTriggerRenderer.getTitleAndSubtitle({ event: artifactEvent("gcp.onArtifactPush", pushData) }).title,
    ).toBe(pushTitle);
    expect(
      onArtifactAnalysisTriggerRenderer.getTitleAndSubtitle({
        event: artifactEvent("gcp.onArtifactAnalysis", analysisData),
      }).title,
    ).toBe(analysisTitle);
  });

  it.each<[string | undefined, string]>([
    ["host/project/repository/image@sha256:abcdef123456", "Pushed image@abcdef12"],
    ["invalid", "Pushed image:v1"],
    [undefined, "Pushed image:v1"],
  ])("prefers a valid push digest and otherwise uses the tag (%j)", (digest, title) => {
    const data: ArtifactPushData = { digest, tag: "host/project/repository/image:v1" };
    expect(
      onArtifactPushTriggerRenderer.getTitleAndSubtitle({ event: artifactEvent("gcp.onArtifactPush", data) }).title,
    ).toBe(title);
  });
});

describe("getArtifactMapper.getExecutionDetails", () => {
  it("maps ArtifactVersionData fields", () => {
    const createTime = "2026-01-01T10:00:00Z";
    const updateTime = "2026-01-01T11:00:00Z";
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "projects/p/locations/us/repositories/r/dockerImages/img@sha256:abcdef",
              createTime,
              updateTime,
              metadata: {
                name: "projects/p/locations/us/repositories/r/dockerImages/img@sha256:abcdef",
                imageSizeBytes: "2048",
              },
            }),
          ],
        },
      },
    });
    const details = getArtifactMapper.getExecutionDetails(ctx);
    expect(details["Image"]).toBe("https://us-docker.pkg.dev/p/r/img@sha256:abcdef");
    expect(details["Image Created At"]).toBe(new Date(createTime).toLocaleString());
    expect(details["Image Updated At"]).toBe(new Date(updateTime).toLocaleString());
    expect(details["Size"]).toBe("2.0 KB");
    expect(details["Digest"]).toBe("img@sha256:abcdef");
  });
});

describe("getArtifactAnalysisMapper.getExecutionDetails", () => {
  it("maps GetArtifactAnalysisData fields", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              resourceUri: "us-docker.pkg.dev/p/r/img@sha256:abc",
              scanStatus: "FINISHED",
              vulnerabilities: 4,
              critical: 1,
              high: 2,
              fixAvailable: 3,
            }),
          ],
        },
      },
    });
    const details = getArtifactAnalysisMapper.getExecutionDetails(ctx);
    expect(details["Image"]).toBe("us-docker.pkg.dev/p/r/img@sha256:abc");
    expect(details["Scan Status"]).toBe("FINISHED");
    expect(details["Vulnerabilities"]).toBe("4");
    expect(details["Critical"]).toBe("1");
    expect(details["High"]).toBe("2");
    expect(details["Fixes Available"]).toBe("3");
  });
});
