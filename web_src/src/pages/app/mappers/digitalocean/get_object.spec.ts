import { describe, expect, it } from "bun:test";
import { getObjectMapper } from "./get_object";
import { buildDetailsCtx, buildOutput } from "./gpu_droplet_test_helpers";
import type { GetObjectOutput } from "./types";

const foundObject: GetObjectOutput = {
  filePath: "reports/summary.txt",
  endpoint: "https://reports.nyc3.digitaloceanspaces.com",
  contentType: "text/plain",
  size: "1024",
  lastModified: new Date().toISOString(),
  tags: { team: "platform", environment: "production" },
  metadata: { owner: "operations", category: "report" },
};

describe("getObjectMapper.getExecutionDetails", () => {
  for (const createdAt of [new Date().toISOString(), undefined]) {
    const timestampDetails = createdAt ? { "Executed At": new Date(createdAt).toLocaleString() } : {};

    describe(createdAt ? "with a timestamp" : "without a timestamp", () => {
      it("formats the first found object and preserves field and entry order", () => {
        const ctx = buildDetailsCtx({
          execution: {
            createdAt,
            outputs: { found: [buildOutput(foundObject), buildOutput({ filePath: "ignored.txt" })] },
          },
        });

        expect(Object.entries(getObjectMapper.getExecutionDetails(ctx))).toEqual([
          ...Object.entries(timestampDetails),
          ["File Path", foundObject.filePath],
          ["Origin Endpoint", foundObject.endpoint],
          ["Content Type", foundObject.contentType],
          ["Size", foundObject.size],
          ["Last Modified", foundObject.lastModified],
          ["Tags", "team=platform, environment=production"],
          ["Metadata", "owner=operations, category=report"],
        ]);
      });

      it.each([
        ["absent values", {}],
        [
          "empty values",
          { filePath: "", endpoint: "", contentType: "", size: "", lastModified: "", tags: {}, metadata: {} },
        ],
      ])("uses dash fallbacks and omits Tags and Metadata for %s", (_name, result) => {
        const ctx = buildDetailsCtx({ execution: { createdAt, outputs: { found: [buildOutput(result)] } } });

        expect(getObjectMapper.getExecutionDetails(ctx)).toEqual({
          ...timestampDetails,
          "File Path": "-",
          "Origin Endpoint": "-",
          "Content Type": "-",
          Size: "-",
          "Last Modified": "-",
        });
      });

      it.each([
        ["notFound", { notFound: [buildOutput({ filePath: "missing.txt" })] }, "missing.txt"],
        ["not_found", { not_found: [buildOutput({ filePath: "missing.txt" })] }, "missing.txt"],
        [
          "both aliases",
          {
            notFound: [buildOutput({ filePath: "preferred.txt" }), buildOutput({ filePath: "ignored.txt" })],
            not_found: [buildOutput({ filePath: "fallback.txt" })],
          },
          "preferred.txt",
        ],
        [
          "empty notFound array",
          { notFound: [], not_found: [buildOutput({ filePath: "fallback.txt" })] },
          "fallback.txt",
        ],
        ["missing file path", { notFound: [buildOutput({})] }, "-"],
        ["empty file path", { not_found: [buildOutput({ filePath: "" })] }, "-"],
      ])("gives not-found output precedence for %s", (_name, outputs, filePath) => {
        const ctx = buildDetailsCtx({
          execution: { createdAt, outputs: { found: [buildOutput(foundObject)], ...outputs } },
        });

        expect(getObjectMapper.getExecutionDetails(ctx)).toEqual({
          ...timestampDetails,
          "File Path": filePath,
          Result: "Not Found",
        });
      });

      it.each([undefined, {}, { found: [] }])(
        "returns only timestamp details when output is missing: %j",
        (outputs) => {
          const ctx = buildDetailsCtx({ execution: { createdAt, outputs } });

          expect(getObjectMapper.getExecutionDetails(ctx)).toEqual(timestampDetails);
        },
      );
    });
  }
});
