import { describe, expect, it } from "vitest";

import {
  centsToDollarInput,
  clearHostedCreditGrantSnapshot,
  dollarInputToCents,
  hostedCreditRefreshMessage,
  hostedCreditRefreshStatus,
  parseDollarInputToCents,
  readHostedCreditGrantSnapshot,
  rememberHostedCreditGrantSnapshot,
} from "./hostedCredit";

describe("parseDollarInputToCents", () => {
  it("returns null for empty or invalid input", () => {
    expect(parseDollarInputToCents("")).toBeNull();
    expect(parseDollarInputToCents("   ")).toBeNull();
    expect(parseDollarInputToCents("abc")).toBeNull();
    expect(parseDollarInputToCents("-1")).toBeNull();
  });

  it("parses zero and positive dollar amounts", () => {
    expect(parseDollarInputToCents("0")).toBe(0);
    expect(parseDollarInputToCents("0.00")).toBe(0);
    expect(parseDollarInputToCents("50")).toBe(5000);
    expect(parseDollarInputToCents("12.34")).toBe(1234);
  });
});

describe("dollarInputToCents", () => {
  it("maps empty or invalid input to zero", () => {
    expect(dollarInputToCents("")).toBe(0);
    expect(dollarInputToCents("abc")).toBe(0);
  });
});

describe("centsToDollarInput", () => {
  it("formats cents as a two-decimal dollar string", () => {
    expect(centsToDollarInput(0)).toBe("0.00");
    expect(centsToDollarInput(2500)).toBe("25.00");
  });
});

describe("hosted credit return refresh", () => {
  it("stores and reads a grant snapshot", () => {
    rememberHostedCreditGrantSnapshot("org-1", 2500);
    expect(readHostedCreditGrantSnapshot("org-1")).toBe(2500);
    clearHostedCreditGrantSnapshot("org-1");
    expect(readHostedCreditGrantSnapshot("org-1")).toBeNull();
  });

  it("stays refreshing until the grant total increases", () => {
    expect(
      hostedCreditRefreshStatus({
        creditAddedQuery: true,
        snapshotCents: 2500,
        grantTotalCents: 2500,
        timedOut: false,
      }),
    ).toBe("refreshing");
    expect(
      hostedCreditRefreshStatus({
        creditAddedQuery: true,
        snapshotCents: 2500,
        grantTotalCents: 5000,
        timedOut: false,
      }),
    ).toBe("added");
  });

  it("reports a pending update after the refresh timeout", () => {
    expect(
      hostedCreditRefreshStatus({
        creditAddedQuery: true,
        snapshotCents: 2500,
        grantTotalCents: 2500,
        timedOut: true,
      }),
    ).toBe("pending");
    expect(hostedCreditRefreshMessage("refreshing")).toBe("Refreshing hosted credit totals.");
    expect(hostedCreditRefreshMessage("added")).toBe("Hosted credit was added.");
    expect(hostedCreditRefreshMessage("pending")).toBe(
      "Hosted credit is still updating. Refresh the page to see new totals.",
    );
  });
});
