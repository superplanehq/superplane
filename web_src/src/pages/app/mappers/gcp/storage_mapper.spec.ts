import { describe, expect, it } from "vitest";
import { createBucketMapper, getBucketMapper, deleteBucketMapper } from "./storage_mapper";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("storage bucket mappers getExecutionDetails", () => {
  it("createBucket surfaces the created bucket with the timestamp first and a console link", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "my-bucket",
              location: "US",
              locationType: "multi-region",
              storageClass: "STANDARD",
              consoleUrl: "https://console.cloud.google.com/storage/browser/my-bucket",
            }),
          ],
        },
      },
    });
    const details = createBucketMapper.getExecutionDetails(ctx);
    // Timestamp first, then at most a handful of fields total.
    expect(Object.keys(details)[0]).toBe("Completed At");
    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
    expect(details["Bucket"]).toBe("my-bucket");
    expect(details["Location"]).toBe("US");
    expect(details["Storage Class"]).toBe("STANDARD");
    expect(details["Console"]).toBe("https://console.cloud.google.com/storage/browser/my-bucket");
  });

  it("getBucket surfaces the fetched bucket details", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ name: "my-bucket", location: "EU", storageClass: "NEARLINE" })],
        },
      },
    });
    const details = getBucketMapper.getExecutionDetails(ctx);
    expect(details["Bucket"]).toBe("my-bucket");
    expect(details["Location"]).toBe("EU");
    expect(details["Storage Class"]).toBe("NEARLINE");
  });

  it("deleteBucket confirms the deletion", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ name: "my-bucket", deleted: true })] } },
    });
    const details = deleteBucketMapper.getExecutionDetails(ctx);
    expect(details["Bucket"]).toBe("my-bucket");
    expect(details["Deleted"]).toBe("true");
  });

  it("does not throw when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getBucketMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

describe("storage bucket mappers props metadata", () => {
  function propsCtx(configuration: Record<string, unknown>) {
    return {
      node: {
        id: "n1",
        name: "Create Bucket",
        componentName: "gcp.storage.createBucket",
        isCollapsed: false,
        configuration,
        metadata: {},
      },
      nodes: [],
      lastExecutions: [],
      componentDefinition: { name: "gcp.storage.createBucket", label: "Create Bucket", icon: "database" },
    } as unknown as Parameters<typeof createBucketMapper.props>[0];
  }

  it("shows the bucket name and location as chips", () => {
    const props = createBucketMapper.props(propsCtx({ name: "my-bucket", location: "US" }));
    expect(props.metadata?.some((m) => m.label === "my-bucket")).toBe(true);
    expect(props.metadata?.some((m) => m.label === "US")).toBe(true);
  });

  it("hides unresolved expression values instead of rendering them raw", () => {
    const props = getBucketMapper.props(propsCtx({ bucket: "{{ $.inputs.bucket }}" }));
    expect(props.metadata?.length).toBe(0);
  });
});
