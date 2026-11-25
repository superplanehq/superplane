import { describe, expect, it } from "bun:test";

import { hetznerBaseMapper } from "./base";
import type { ComponentBaseContext, ExecutionInfo, NodeInfo } from "../types";

function buildContext(
  configuration?: NodeInfo["configuration"],
  lastExecutions: ExecutionInfo[] = [],
): ComponentBaseContext {
  return {
    nodes: [],
    node: {
      id: "node-1",
      name: "Create server",
      componentName: "hetzner.createServer",
      isCollapsed: false,
      configuration,
    },
    componentDefinition: {
      name: "hetzner.createServer",
      label: "Create Server",
      description: "",
      icon: "server",
      color: "gray",
    },
    lastExecutions,
    currentUser: { id: "user-1", name: "Test User", email: "test@example.com", roles: [], groups: [] },
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

function buildExecution(outputs?: ExecutionInfo["outputs"]): ExecutionInfo {
  const timestamp = new Date().toISOString();
  return {
    id: "execution-1",
    createdAt: timestamp,
    updatedAt: timestamp,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: { id: "event-1", createdAt: timestamp, data: {}, nodeId: "trigger-1", type: "test" },
    outputs,
  };
}

describe("Hetzner metadata", () => {
  it("preserves the complete ordered icon-label list", () => {
    const context = buildContext({
      serverType: " cx22 ",
      image: "ubuntu-24.04",
      description: "Daily backup",
      snapshot: 123,
      location: { label: " Falkenstein ", id: 1 },
      firewall: { name: "Web firewall" },
      server: 0,
      loadBalancer: { id: 42 },
      loadBalancerType: "lb11",
      algorithm: "round_robin",
      sshKeys: ["key-1", null, ""],
    });

    expect(hetznerBaseMapper.props(context).metadata).toEqual([
      { icon: "cpu", label: "Type: cx22" },
      { icon: "hard-drive", label: "Image: ubuntu-24.04" },
      { icon: "camera", label: "Snapshot: Daily backup" },
      { icon: "hard-drive", label: "Snapshot image: 123" },
      { icon: "map-pin", label: "Location: Falkenstein" },
      { icon: "shield", label: "Firewall: Web firewall" },
      { icon: "server", label: "Server: 0" },
      { icon: "route", label: "Load Balancer: 42" },
      { icon: "cpu", label: "LB Type: lb11" },
      { icon: "shuffle", label: "Algorithm: round_robin" },
      { icon: "key", label: "SSH keys: 3" },
    ]);
  });

  it.each([
    { value: " cx22 ", expected: "cx22" },
    { value: 0, expected: "0" },
    {
      value: { label: " Label ", displayName: "Display", name: "Name", slug: "Slug", value: "Value", id: 1 },
      expected: "Label",
    },
    { value: { label: " ", displayName: " Display ", name: "Name" }, expected: "Display" },
    { value: { displayName: false, name: " Name ", slug: "Slug" }, expected: "Name" },
    { value: { name: {}, slug: " Slug ", value: "Value" }, expected: "Slug" },
    { value: { slug: null, value: 0, id: 1 }, expected: "0" },
    { value: { value: "", id: " ID " }, expected: "ID" },
  ])("normalizes $value to $expected", ({ value, expected }) => {
    expect(hetznerBaseMapper.props(buildContext({ serverType: value })).metadata).toEqual([
      { icon: "cpu", label: `Type: ${expected}` },
    ]);
  });

  it.each(
    [undefined, null, "", " \t ", false, true, {}, { label: " ", id: null }, [], ["cx22"]].map((value) => ({ value })),
  )("omits empty or unsupported configuration values: $value", ({ value }) => {
    expect(hetznerBaseMapper.props(buildContext({ serverType: value })).metadata).toEqual([]);
  });

  it("omits metadata when configuration is missing", () => {
    expect(hetznerBaseMapper.props(buildContext()).metadata).toEqual([]);
  });

  it.each([
    { image: "123", outputName: " Ubuntu ", expected: "Image: Ubuntu (123)" },
    { image: " Ubuntu ", outputName: "Ubuntu", expected: "Image: Ubuntu" },
    { image: "123", outputName: undefined, expected: "Image: 123" },
    { image: "123", outputName: " ", expected: "Image: 123" },
    { image: "123", outputName: { label: " Ubuntu " }, expected: "Image: Ubuntu (123)" },
    { image: " ", outputName: "Ubuntu", expected: undefined },
    { image: undefined, outputName: "Ubuntu", expected: undefined },
  ])("formats configured image $image with output name $outputName", ({ image, outputName, expected }) => {
    const execution = buildExecution({ default: [{ data: { imageName: outputName } }] });
    expect(hetznerBaseMapper.props(buildContext({ image }, [execution])).metadata).toEqual(
      expected ? [{ icon: "hard-drive", label: expected }] : [],
    );
  });

  it("uses only the latest execution's first default output for the image name", () => {
    const latest = buildExecution({
      default: [{ data: { imageName: "Latest" } }, { data: { imageName: "Second" } }],
      other: [{ data: { imageName: "Other channel" } }],
    });
    const older = buildExecution({ default: [{ data: { imageName: "Older" } }] });
    expect(hetznerBaseMapper.props(buildContext({ image: "123" }, [latest, older])).metadata).toEqual([
      { icon: "hard-drive", label: "Image: Latest (123)" },
    ]);
  });

  it.each([undefined, {}, { default: [] }, { default: [{}] }])(
    "falls back when output data is missing: %j",
    (outputs) => {
      expect(hetznerBaseMapper.props(buildContext({ image: "123" }, [buildExecution(outputs)])).metadata).toEqual([
        { icon: "hard-drive", label: "Image: 123" },
      ]);
    },
  );

  it.each([[], "key-1", { 0: "key-1", length: 1 }, null, undefined].map((sshKeys) => ({ sshKeys })))(
    "omits SSH key counts for $sshKeys",
    ({ sshKeys }) => {
      expect(hetznerBaseMapper.props(buildContext({ sshKeys })).metadata).toEqual([]);
    },
  );
});
