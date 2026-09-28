import { describe, expect, it } from "bun:test";

import { lineAppIds, lineUsesHostedCredit, runnerUsesHostedCredit } from "./lineHostedCredit";

describe("runnerUsesHostedCredit", () => {
  it("treats SuperPlane Agent and hosted credentials as hosted credit", () => {
    expect(runnerUsesHostedCredit({ component: "runnerSuperPlane" })).toBe(true);
    expect(
      runnerUsesHostedCredit({
        component: "runnerClaudeCode",
        configuration: { credentials: { source: "hosted" } },
      }),
    ).toBe(true);
    expect(runnerUsesHostedCredit({ component: "runnerCodex" })).toBe(true);
  });

  it("treats a customer key as BYOK", () => {
    expect(
      runnerUsesHostedCredit({
        component: "runnerClaudeCode",
        configuration: { credentials: { source: "integration" } },
      }),
    ).toBe(false);
    expect(
      runnerUsesHostedCredit({
        component: "runnerOpenRouter",
        configuration: { credentials: { source: "secret" } },
      }),
    ).toBe(false);
  });

  it("ignores nodes that are not agent runners", () => {
    expect(runnerUsesHostedCredit({ component: "runnerBash" })).toBeNull();
  });
});

describe("lineUsesHostedCredit", () => {
  it("is false when every runner uses a customer key", () => {
    expect(
      lineUsesHostedCredit([
        { component: "runnerClaudeCode", configuration: { credentials: { source: "integration" } } },
        { component: "runnerBash" },
      ]),
    ).toBe(false);
  });

  it("is true when any runner spends hosted credit", () => {
    expect(
      lineUsesHostedCredit([
        { component: "runnerClaudeCode", configuration: { credentials: { source: "integration" } } },
        { component: "runnerSuperPlane" },
      ]),
    ).toBe(true);
  });
});

describe("lineAppIds", () => {
  it("returns unique app ids from line steps", () => {
    expect(
      lineAppIds({
        steps: [{ app: { app: "app-1" } }, { app: { app: "app-1" } }, { app: { app: "app-2" } }, {}],
      }),
    ).toEqual(["app-1", "app-2"]);
  });
});
