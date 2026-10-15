import { describe, expect, it } from "bun:test";
import type { ComponentDefinition } from "../types";
import { purgeCacheMapper } from "./purge_cache";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.purgeCache",
  label: "Purge Cache",
  description: "",
  icon: "zap",
  color: "orange",
};

const {
  buildNode,
  buildDetailsCtx,
  buildPropsContext: buildPropsCtx,
} = createComponentFixtures(defaultDefinition, "Purge CDN Cache");
const buildOutput = (data: unknown) => buildCloudflareOutput("cloudflare.cache.purged", data);

// ── props ─────────────────────────────────────────────────────────────────────

describe("purgeCacheMapper.props metadata", () => {
  it("shows purge everything label when mode is everything", () => {
    const props = purgeCacheMapper.props(buildPropsCtx({ node: buildNode({ configuration: { mode: "everything" } }) }));
    expect(props.metadata).toEqual([{ icon: "zap", label: "Purge everything" }]);
  });

  it("shows URL count when mode is files", () => {
    const props = purgeCacheMapper.props(
      buildPropsCtx({
        node: buildNode({
          configuration: {
            mode: "files",
            files: ["https://example.com/a.js", "https://example.com/b.css"],
          },
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "link", label: "2 URLs" }]);
  });

  it("shows singular URL label for single file", () => {
    const props = purgeCacheMapper.props(
      buildPropsCtx({
        node: buildNode({ configuration: { mode: "files", files: ["https://example.com/a.js"] } }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "link", label: "1 URL" }]);
  });

  it("shows tag count when mode is tags", () => {
    const props = purgeCacheMapper.props(
      buildPropsCtx({
        node: buildNode({ configuration: { mode: "tags", tags: ["v1.2.3", "static-assets"] } }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "tag", label: "2 tags" }]);
  });

  it("shows host count when mode is hosts", () => {
    const props = purgeCacheMapper.props(
      buildPropsCtx({
        node: buildNode({ configuration: { mode: "hosts", hosts: ["preview.example.com"] } }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "server", label: "1 host" }]);
  });

  it("shows prefix count when mode is prefixes", () => {
    const props = purgeCacheMapper.props(
      buildPropsCtx({
        node: buildNode({ configuration: { mode: "prefixes", prefixes: ["www.example.com/foo"] } }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "folder-tree", label: "1 prefix" }]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = purgeCacheMapper.props(buildPropsCtx({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("purgeCacheMapper.getExecutionDetails", () => {
  it("returns details with mode, zone, and file count", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              zoneId: "zone123",
              zoneName: "example.com",
              id: "purge-abc",
              mode: "files",
              files: ["https://example.com/a.js", "https://example.com/b.js"],
              prefixes: ["www.example.com/foo"],
            }),
          ],
        },
      },
    });
    const details = purgeCacheMapper.getExecutionDetails(ctx);
    expect(details["Mode"]).toBe("files");
    expect(details["Zone"]).toBe("example.com");
    expect(details["Files"]).toBe("2");
    expect(details["Prefixes"]).toBe("1");
  });

  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => purgeCacheMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("falls back to zone ID when zone name is omitted", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              zoneId: "zone123",
              id: "p",
              mode: "everything",
            }),
          ],
        },
      },
    });
    expect(purgeCacheMapper.getExecutionDetails(ctx)["Zone"]).toBe("zone123");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ zoneId: "z", id: "p", mode: "everything" })] } },
    });
    expect(purgeCacheMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.purgeCache", () => {
  it("maps finished success to purged", () => {
    expect(eventStateRegistry.purgeCache.getState(buildExecution())).toBe("purged");
  });
});
