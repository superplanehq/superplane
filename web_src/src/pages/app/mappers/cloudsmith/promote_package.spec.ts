import { describe, expect, it } from "vitest";
import { promotePackageMapper, promotePackageEventStateRegistry } from "./promote_package";
import { buildDetailsCtx, buildPackageData, buildPackageOutput, buildNode } from "./test_helpers";

describe("promotePackageMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => promotePackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => promotePackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At without package fields when output data is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPromoteOutput(undefined)] } },
    });
    const details = promotePackageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Package"]).toBeUndefined();
  });

  it("extracts key promoted package fields", () => {
    const pkg = buildPackageData({
      name: "my-package",
      version: "1.2.0",
      repository: "production",
    });
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPromoteOutput(pkg)] } },
    });
    const details = promotePackageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Package"]).toBe("my-package");
    expect(details["Version"]).toBe("1.2.0");
    expect(details["Destination"]).toBe("production");
  });

  it("does not include Status, URL, Size, or Security Scan in details", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPromoteOutput(buildPackageData())] } },
    });
    const details = promotePackageMapper.getExecutionDetails(ctx);
    expect(details["Status"]).toBeUndefined();
    expect(details["URL"]).toBeUndefined();
    expect(details["Size"]).toBeUndefined();
    expect(details["Security Scan"]).toBeUndefined();
  });
});

describe("promotePackageEventStateRegistry.getState", () => {
  const { getState } = promotePackageEventStateRegistry;

  it("returns copied when mode is copy (default)", () => {
    const execution = buildExecution({
      result: "RESULT_PASSED",
      state: "STATE_FINISHED",
      configuration: { mode: "copy" },
    });
    expect(getState(execution)).toBe("copied");
  });

  it("returns copied when mode is not set", () => {
    const execution = buildExecution({ result: "RESULT_PASSED", state: "STATE_FINISHED", configuration: {} });
    expect(getState(execution)).toBe("copied");
  });

  it("returns moved when mode is move", () => {
    const execution = buildExecution({
      result: "RESULT_PASSED",
      state: "STATE_FINISHED",
      configuration: { mode: "move" },
    });
    expect(getState(execution)).toBe("moved");
  });

  it("returns failed for a failed execution", () => {
    const execution = buildExecution({
      result: "RESULT_FAILED",
      state: "STATE_FINISHED",
      configuration: { mode: "move" },
    });
    expect(getState(execution)).toBe("failed");
  });

  it("has stateMap entries for both copied and moved", () => {
    expect(promotePackageEventStateRegistry.stateMap["copied"]).toBeDefined();
    expect(promotePackageEventStateRegistry.stateMap["moved"]).toBeDefined();
  });
});

describe("promotePackageMapper metadata", () => {
  it("shows destination repository from configuration", () => {
    const node = buildNode({
      configuration: {
        sourceRepository: "acme/staging",
        package: "perm123",
        destinationRepository: "acme/production",
        mode: "copy",
      },
      metadata: {},
    });
    const metadata =
      promotePackageMapper.props({
        nodes: [node],
        node,
        componentDefinition: {
          name: "cloudsmith.promotePackage",
          label: "Promote Package",
          description: "",
          icon: "copy",
          color: "blue",
        },
        lastExecutions: [],
      }).metadata ?? [];
    const labels = metadata.map((m) => m.label);
    expect(labels).toContain("acme/production");
  });

  it("shows action label from configuration mode", () => {
    const node = buildNode({
      configuration: {
        sourceRepository: "acme/staging",
        package: "perm123",
        destinationRepository: "acme/production",
        mode: "move",
      },
      metadata: {},
    });
    const metadata =
      promotePackageMapper.props({
        nodes: [node],
        node,
        componentDefinition: {
          name: "cloudsmith.promotePackage",
          label: "Promote Package",
          description: "",
          icon: "copy",
          color: "blue",
        },
        lastExecutions: [],
      }).metadata ?? [];
    const labels = metadata.map((m) => m.label);
    expect(labels).toContain("Move");
  });

  it("shows Copy for copy mode", () => {
    const node = buildNode({
      configuration: {
        sourceRepository: "acme/staging",
        package: "perm123",
        destinationRepository: "acme/production",
        mode: "copy",
      },
      metadata: {},
    });
    const metadata =
      promotePackageMapper.props({
        nodes: [node],
        node,
        componentDefinition: {
          name: "cloudsmith.promotePackage",
          label: "Promote Package",
          description: "",
          icon: "copy",
          color: "blue",
        },
        lastExecutions: [],
      }).metadata ?? [];
    const labels = metadata.map((m) => m.label);
    expect(labels).toContain("Copy");
  });
});

function buildPromoteOutput(data: unknown) {
  return buildPackageOutput(data, "cloudsmith.package.promoted");
}

function buildExecution(overrides: { result: string; state: string; configuration: unknown }) {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: overrides.state as never,
    result: overrides.result as never,
    resultReason: "RESULT_REASON_OK" as never,
    resultMessage: "",
    metadata: {},
    configuration: overrides.configuration,
    rootEvent: undefined as never,
  };
}
