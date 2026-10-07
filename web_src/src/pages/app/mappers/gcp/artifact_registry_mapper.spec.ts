import { describe, expect, it } from "bun:test";

import { getArtifactAnalysisMapper, getArtifactMapper } from "./artifact_registry_mapper";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

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
