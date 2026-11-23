import { describe, expect, it } from "bun:test";
import { getDropletMapper } from "./get_droplet";
import { buildDetailsCtx, buildDropletData, buildOutput } from "./gpu_droplet_test_helpers";

describe("getDropletMapper.getExecutionDetails", () => {
  it("formats full details in display order using the first output and public IPv4", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput(
              buildDropletData({
                networks: {
                  v4: [
                    { type: "private", ip_address: "10.0.0.1" },
                    { type: "public", ip_address: "1.2.3.4" },
                    { type: "public", ip_address: "5.6.7.8" },
                  ],
                },
                tags: ["web", "prod"],
              }),
            ),
            buildOutput(buildDropletData({ id: 999, name: "later-droplet" })),
          ],
        },
      },
    });
    const details = getDropletMapper.getExecutionDetails(ctx);
    expect(details).toEqual({
      "Executed At": new Date(ctx.execution.createdAt!).toLocaleString(),
      "Droplet ID": "123456",
      Name: "gpu-droplet-1",
      Status: "active",
      Region: "New York 3",
      Size: "gpu-h100x1-80gb",
      OS: "Ubuntu 22.04 (LTS) x64",
      Memory: "245760 MB",
      vCPUs: "20",
      Disk: "480 GB",
      "IP Address": "1.2.3.4",
      Tags: "web, prod",
    });
    expect(Object.keys(details)).toEqual([
      "Executed At",
      "Droplet ID",
      "Name",
      "Status",
      "Region",
      "Size",
      "OS",
      "Memory",
      "vCPUs",
      "Disk",
      "IP Address",
      "Tags",
    ]);
  });

  it.each([
    {
      label: "name precedence",
      region: { name: "New York 3", slug: "nyc3" },
      image: { name: "Ubuntu", slug: "ubuntu" },
      expected: ["New York 3", "Ubuntu"],
    },
    {
      label: "slug fallback",
      region: { name: "", slug: "nyc3" },
      image: { name: "", slug: "ubuntu" },
      expected: ["nyc3", "ubuntu"],
    },
    { label: "empty objects", region: {}, image: {}, expected: ["-", "-"] },
    { label: "missing objects", region: undefined, image: undefined, expected: ["-", "-"] },
  ])("formats sparse details with $label", ({ region, image, expected }) => {
    const ctx = buildDetailsCtx({
      execution: {
        createdAt: undefined,
        outputs: { default: [buildOutput({ region, image, tags: [] })] },
      },
    });
    expect(getDropletMapper.getExecutionDetails(ctx)).toEqual({
      "Droplet ID": "-",
      Name: "-",
      Status: "-",
      Region: expected[0],
      Size: "-",
      OS: expected[1],
      Memory: "-",
      vCPUs: "-",
      Disk: "-",
    });
  });

  it.each([
    { label: "numeric", zero: 0, memory: "-", disk: "-" },
    { label: "string", zero: "0", memory: "0 MB", disk: "0 GB" },
  ])("preserves $label zero hardware formatting", ({ zero, memory, disk }) => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildDropletData({ memory: zero, vcpus: zero, disk: zero }))] } },
    });
    const details = getDropletMapper.getExecutionDetails(ctx);
    expect(details.Memory).toBe(memory);
    expect(details.vCPUs).toBe("0");
    expect(details.Disk).toBe(disk);
  });

  it.each([
    { label: "missing outputs", outputs: undefined },
    { label: "missing default output", outputs: {} },
    { label: "empty default outputs", outputs: { default: [] } },
    { label: "missing first payload", outputs: { default: [buildOutput(undefined), buildOutput(buildDropletData())] } },
  ])("returns only the available timestamp for $label", ({ outputs }) => {
    const ctx = buildDetailsCtx({ execution: { outputs } });
    expect(getDropletMapper.getExecutionDetails(ctx)).toEqual({
      "Executed At": new Date(ctx.execution.createdAt!).toLocaleString(),
    });
    const ctxWithoutTimestamp = buildDetailsCtx({ execution: { outputs, createdAt: undefined } });
    expect(getDropletMapper.getExecutionDetails(ctxWithoutTimestamp)).toEqual({});
  });

  it("omits Tags when tags is a truthy non-array value", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildDropletData({ tags: "web" }))] } },
    });
    expect(() => getDropletMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(getDropletMapper.getExecutionDetails(ctx)["Tags"]).toBeUndefined();
  });
});
