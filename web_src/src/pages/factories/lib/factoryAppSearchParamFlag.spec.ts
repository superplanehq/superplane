import { describe, expect, it } from "bun:test";

import {
  leaveFactoryConfigureSearchParams,
  setFactoryAppSelectedRun,
  setSearchParamFlag,
} from "./factoryAppSearchParamFlag";

describe("leaveFactoryConfigureSearchParams", () => {
  it("drops Configure chrome flags and keeps run context", () => {
    const next = leaveFactoryConfigureSearchParams(
      new URLSearchParams("configure=1&agent=1&blocks=1&from=lines&lineId=l1&run=r1"),
    );
    expect(next.get("configure")).toBeNull();
    expect(next.get("agent")).toBeNull();
    expect(next.get("blocks")).toBeNull();
    expect(next.get("from")).toBe("lines");
    expect(next.get("lineId")).toBe("l1");
    expect(next.get("run")).toBe("r1");
  });

  it("returns the same params when Configure flags are already absent", () => {
    const current = new URLSearchParams("from=automations&run=r1");
    expect(leaveFactoryConfigureSearchParams(current)).toBe(current);
  });
});

describe("setFactoryAppSelectedRun", () => {
  it("sets the run id and keeps other flags", () => {
    const next = setFactoryAppSelectedRun(new URLSearchParams("from=lines&lineId=l1"), "run-9");
    expect(next.get("run")).toBe("run-9");
    expect(next.get("from")).toBe("lines");
    expect(next.get("lineId")).toBe("l1");
  });

  it("clears the run id", () => {
    const next = setFactoryAppSelectedRun(new URLSearchParams("from=automations&run=run-9"), null);
    expect(next.get("run")).toBeNull();
    expect(next.get("from")).toBe("automations");
  });

  it("returns the same params when the run id already matches", () => {
    const current = new URLSearchParams("run=run-9");
    expect(setFactoryAppSelectedRun(current, "run-9")).toBe(current);
  });
});

describe("setSearchParamFlag", () => {
  it("sets the flag to 1 when open is true", () => {
    const next = setSearchParamFlag(new URLSearchParams("from=lines"), "yaml", true);
    expect(next.get("yaml")).toBe("1");
    expect(next.get("from")).toBe("lines");
  });

  it("removes the flag when open is false", () => {
    const next = setSearchParamFlag(new URLSearchParams("from=lines&yaml=1"), "yaml", false);
    expect(next.get("yaml")).toBeNull();
    expect(next.get("from")).toBe("lines");
  });

  it("returns the same params when the flag already matches", () => {
    const current = new URLSearchParams("yaml=1");
    expect(setSearchParamFlag(current, "yaml", true)).toBe(current);
  });
});
