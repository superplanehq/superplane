import { describe, expect, it } from "vitest";

import { setSearchParamFlag } from "./factoryAppSearchParamFlag";

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
