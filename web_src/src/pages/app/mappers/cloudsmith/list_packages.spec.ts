import { describe, expect, it } from "vitest";
import { listPackagesMapper } from "./list_packages";
import { buildDetailsCtx } from "./test_helpers";
import type { TrimmedPackageData } from "./types";

describe("listPackagesMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => listPackagesMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => listPackagesMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At when packages are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildListPackagesOutput(undefined)] } },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Packages Found"]).toBeUndefined();
  });

  it("shows total packages found count", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildListPackagesOutput([buildTrimmedPackage(), buildTrimmedPackage()])],
        },
      },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Packages Found"]).toBe("2");
  });

  it("shows quarantined package count", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildListPackagesOutput([
              buildTrimmedPackage({ is_quarantined: true }),
              buildTrimmedPackage({ is_quarantined: false }),
              buildTrimmedPackage({ is_quarantined: true }),
            ]),
          ],
        },
      },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Quarantined"]).toBe("2");
  });

  it("shows vulnerable (policy_violated) package count", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildListPackagesOutput([
              buildTrimmedPackage({ policy_violated: true }),
              buildTrimmedPackage({ policy_violated: false }),
            ]),
          ],
        },
      },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Vulnerable"]).toBe("1");
  });

  it("shows zero quarantined and vulnerable when all packages are clean", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildListPackagesOutput([buildTrimmedPackage(), buildTrimmedPackage()])],
        },
      },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Quarantined"]).toBe("0");
    expect(details["Vulnerable"]).toBe("0");
  });

  it("does not include Format, Status, Security Scan, or Repository URL in details", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildListPackagesOutput([buildTrimmedPackage({ format: "docker", status_str: "Available" })]),
          ],
        },
      },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Format"]).toBeUndefined();
    expect(details["Status"]).toBeUndefined();
    expect(details["Security Scan"]).toBeUndefined();
    expect(details["Repository URL"]).toBeUndefined();
  });
});

function buildTrimmedPackage(overrides?: Partial<TrimmedPackageData>): TrimmedPackageData {
  return {
    display_name: "my-package",
    format: "docker",
    is_quarantined: false,
    policy_violated: false,
    repository: "production",
    slug_perm: "perm123abc",
    stage_str: "Fully Synchronised",
    status_str: "Available",
    ...overrides,
  };
}

function buildListPackagesOutput(packages: TrimmedPackageData[] | undefined) {
  return {
    type: "cloudsmith.packages.listed",
    timestamp: new Date().toISOString(),
    data: packages !== undefined ? { packages } : undefined,
  };
}
