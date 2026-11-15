import { describe, expect, it } from "bun:test";

import { resolveEmailFirstSso } from "./discover";

describe("resolveEmailFirstSso", () => {
  it("continues when SSO is disabled", async () => {
    const decision = await resolveEmailFirstSso({ email: "ada@example.com", signup: false, ssoEnabled: false });
    expect(decision).toEqual({ kind: "continue" });
  });

  it("continues in signup mode", async () => {
    const decision = await resolveEmailFirstSso({ email: "ada@example.com", signup: true, ssoEnabled: true });
    expect(decision).toEqual({ kind: "continue" });
  });

  it("prompts when the domain uses SSO", async () => {
    globalThis.fetch = (() =>
      Promise.resolve(
        Response.json({ ssoAvailable: true, startUrl: "/auth/sso/1/start", required: true }),
      )) as typeof fetch;
    const decision = await resolveEmailFirstSso({ email: "ada@example.com", signup: false, ssoEnabled: true });
    expect(decision).toEqual({ kind: "prompt", startUrl: "/auth/sso/1/start", required: true });
  });

  it("continues when the domain has no SSO", async () => {
    globalThis.fetch = (() => Promise.resolve(Response.json({ ssoAvailable: false }))) as typeof fetch;
    const decision = await resolveEmailFirstSso({ email: "ada@example.com", signup: false, ssoEnabled: true });
    expect(decision).toEqual({ kind: "continue" });
  });

  it("continues when discovery fails", async () => {
    globalThis.fetch = (() => Promise.reject(new Error("offline"))) as typeof fetch;
    const decision = await resolveEmailFirstSso({ email: "ada@example.com", signup: false, ssoEnabled: true });
    expect(decision).toEqual({ kind: "continue" });
  });

  it("continues when discovery returns an error status", async () => {
    globalThis.fetch = (() => Promise.resolve(new Response("", { status: 429 }))) as typeof fetch;
    const decision = await resolveEmailFirstSso({ email: "ada@example.com", signup: false, ssoEnabled: true });
    expect(decision).toEqual({ kind: "continue" });
  });
});
