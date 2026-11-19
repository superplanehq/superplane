import { describe, expect, it } from "bun:test";
import { createDropletMapper } from "./create_droplet";
import type { DropletPayload } from "./droplet_payloads";
import { buildDetailsCtx, buildOutput } from "./gpu_droplet_test_helpers";

const droplet: DropletPayload = {
  id: 123456,
  name: "web-1",
  region: { name: "New York 3", slug: "nyc3" },
  size_slug: "s-1vcpu-1gb",
  image: { name: "Ubuntu 22.04 (LTS) x64", slug: "ubuntu-22-04-x64" },
  networks: {
    v4: [
      { type: "private", ip_address: "10.0.0.1" },
      { type: "public", ip_address: "203.0.113.1" },
      { type: "public", ip_address: "203.0.113.2" },
    ],
  },
};

describe("createDropletMapper.getExecutionDetails", () => {
  it("returns only the timestamp when output is missing", () => {
    const createdAt = new Date().toISOString();
    const ctx = buildDetailsCtx({ execution: { createdAt, outputs: undefined } });

    expect(createDropletMapper.getExecutionDetails(ctx)).toEqual({
      "Created At": new Date(createdAt).toLocaleString(),
    });
  });

  it("formats droplet fields with names and the first public IPv4 address", () => {
    const ctx = buildDetailsCtx({ execution: { createdAt: undefined, outputs: { default: [buildOutput(droplet)] } } });

    expect(createDropletMapper.getExecutionDetails(ctx)).toEqual({
      "Droplet ID": "123456",
      Name: "web-1",
      Region: "New York 3",
      Size: "s-1vcpu-1gb",
      OS: "Ubuntu 22.04 (LTS) x64",
      "IP Address": "203.0.113.1",
    });
  });

  it("falls back to region and image slugs when names are empty or missing", () => {
    const slugOnlyDroplet: DropletPayload = {
      ...droplet,
      region: { name: "", slug: "nyc3" },
      image: { slug: "ubuntu-22-04-x64" },
    };
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(slugOnlyDroplet)] } } });

    expect(createDropletMapper.getExecutionDetails(ctx)).toMatchObject({
      Region: "nyc3",
      OS: "ubuntu-22-04-x64",
    });
  });

  it("keeps placeholders and omits IP Address when no public IPv4 address exists", () => {
    const incompleteDroplet: DropletPayload = {
      name: "",
      networks: { v4: [{ type: "private", ip_address: "10.0.0.1" }] },
    };
    const ctx = buildDetailsCtx({
      execution: { createdAt: undefined, outputs: { default: [buildOutput(incompleteDroplet)] } },
    });

    expect(createDropletMapper.getExecutionDetails(ctx)).toEqual({
      "Droplet ID": "-",
      Name: "-",
      Region: "-",
      Size: "-",
      OS: "-",
    });
  });
});
