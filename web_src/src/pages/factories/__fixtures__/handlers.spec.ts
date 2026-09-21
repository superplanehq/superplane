import { describe, expect, it } from "bun:test";

import { fetchFactoryPageFixture } from "./handlers";
import { lineMetricsFactoriesFixture } from "./lineMetricsFactoriesFixture";
import { refineChatBoardFixture } from "./refineChatBoardFixture";
import {
  CLOSED_WORK_ORDER,
  defaultFactoriesFixture,
  DRAFT_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  REFUND_LINE_ONBOARDING_ID,
  REFUND_LINE_PLAN_ID,
  RUNNING_WORK_ORDER,
} from "./factoryPageResponses";
import { HEADER_MCP_RESOURCE } from "./agentResourceFixtures";
import { BUSINESS_ORGANIZATION_BILLING } from "./usageReportFixtures";

describe("matchFactoryPageFixture", () => {
  it("lists factories and returns the primary factory by id", async () => {
    const list = await fetchFactoryPageFixture("/api/v1/factories");
    await expect(list.json()).resolves.toMatchObject({
      factories: expect.arrayContaining([expect.objectContaining({ name: "Semaphore" })]),
    });

    const detail = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}`);
    const body = (await detail.json()) as {
      factory: { id?: string; name?: string; lines?: Array<{ id?: string; metrics?: { successRatePct?: number } }> };
    };
    expect(body.factory).toMatchObject({ id: PRIMARY_FACTORY_ID, name: "Semaphore" });
    const plan = body.factory.lines?.find((line) => line.id === REFUND_LINE_PLAN_ID);
    expect(plan?.metrics?.successRatePct).toBe(82);
  });

  it("serves tasks and includes both open and closed entries", async () => {
    const orders = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/orders`);
    const body = (await orders.json()) as { orders: Array<{ id?: string; state?: string }> };
    const ids = body.orders.map((entry) => entry.id);
    expect(ids).toEqual(expect.arrayContaining([OPEN_WORK_ORDER.id, RUNNING_WORK_ORDER.id, CLOSED_WORK_ORDER.id]));
  });

  it("serves factory usage, usage history, and organization workspace usage reports", async () => {
    const usage = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/usage`);
    await expect(usage.json()).resolves.toMatchObject({
      totalTokens: "25600",
      totalCostCents: "876",
      byModel: expect.arrayContaining([expect.objectContaining({ provider: "anthropic" })]),
    });

    const history = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/usage-history`);
    await expect(history.json()).resolves.toMatchObject({
      totalCount: 3,
      rows: expect.arrayContaining([
        expect.objectContaining({
          workOrderKey: "RF-101",
          models: [],
          byokModels: ["anthropic/claude-sonnet-4-6"],
        }),
      ]),
    });

    const spend = await fetchFactoryPageFixture(`/api/v1/organizations/${FACTORIES_ORGANIZATION_ID}/workspace-usage`);
    await expect(spend.json()).resolves.toMatchObject({
      totalTokens: "25600",
      totalCostCents: "876",
    });

    const hosted = await fetchFactoryPageFixture(
      `/api/v1/organizations/${FACTORIES_ORGANIZATION_ID}/hosted-llm-models?provider=anthropic`,
    );
    await expect(hosted.json()).resolves.toMatchObject({
      enabled: true,
      models: [expect.objectContaining({ id: "claude-sonnet-4-6" })],
    });

    const selectable = await fetchFactoryPageFixture(
      `/api/v1/organizations/${FACTORIES_ORGANIZATION_ID}/selectable-llm-models`,
    );
    await expect(selectable.json()).resolves.toMatchObject({
      models: expect.arrayContaining([
        expect.objectContaining({ key: "byok::anthropic::claude-sonnet-4-6" }),
        expect.objectContaining({ key: "hosted::anthropic::claude-sonnet-4-6" }),
      ]),
    });
  });

  it("deletes a factory automation by id", async () => {
    const fixture = structuredClone(defaultFactoriesFixture);
    const created = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/automations`,
      { method: "POST", body: JSON.stringify({ name: "Create env", columnKey: "verify" }) },
      fixture,
    );
    const createdBody = (await created.json()) as { automation?: { id?: string } };
    const automationId = createdBody.automation?.id;
    expect(automationId).toBeTruthy();

    const deleted = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/automations/${automationId}`,
      { method: "DELETE" },
      fixture,
    );
    expect(deleted.status).toBe(200);

    const list = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/automations`,
      undefined,
      fixture,
    );
    const body = (await list.json()) as { automations?: Array<{ id?: string }> };
    expect(body.automations?.some((entry) => entry.id === automationId)).toBe(false);
  });

  it("returns factory automations for the populated factory", async () => {
    const apps = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/automations`);
    await expect(apps.json()).resolves.toMatchObject({
      automations: expect.arrayContaining([expect.objectContaining({ name: "Refund Planner" })]),
    });
  });

  it("includes agent permissions on the factory me user", async () => {
    const me = await fetchFactoryPageFixture("/api/v1/me");
    const body = await me.json();
    expect(body.user.permissions).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ resource: "agents", action: "read" }),
        expect.objectContaining({ resource: "agents", action: "create" }),
      ]),
    );
  });

  it("grants workspace settings permission on /api/v1/me", async () => {
    const me = await fetchFactoryPageFixture("/api/v1/me");
    await expect(me.json()).resolves.toMatchObject({
      user: {
        permissions: expect.arrayContaining([expect.objectContaining({ resource: "factories", action: "update" })]),
      },
    });
  });

  it("omits metrics on an idle onboarding line", async () => {
    const detail = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}`,
      undefined,
      structuredClone(lineMetricsFactoriesFixture),
    );
    const body = (await detail.json()) as {
      factory: { lines?: Array<{ id?: string; metrics?: { successRatePct?: number } }> };
    };
    const onboarding = body.factory.lines?.find((line) => line.id === REFUND_LINE_ONBOARDING_ID);
    expect(onboarding).toBeDefined();
    expect(onboarding?.metrics).toBeUndefined();
  });

  it("creates a workspace with a unique key derived from the name", async () => {
    const response = await fetchFactoryPageFixture("/api/v1/factories", {
      method: "POST",
      body: JSON.stringify({ name: "New workspace", description: "", key: "" }),
    });
    const body = (await response.json()) as { factory?: { id?: string; name?: string; key?: string } };

    expect(body.factory?.name).toBe("New workspace");
    expect(body.factory?.id).toMatch(/^storybook-factory-/);
    expect(body.factory?.key).toBe("NEWWO");
  });

  it("walks to a free key when the name-derived key is already taken", async () => {
    const fixture = structuredClone(defaultFactoriesFixture);
    fixture.factories.push({ id: "taken-newwo", name: "Taken", key: "NEWWO", lines: [] });

    const response = await fetchFactoryPageFixture(
      "/api/v1/factories",
      {
        method: "POST",
        body: JSON.stringify({ name: "New workspace", description: "", key: "" }),
      },
      fixture,
    );
    const body = (await response.json()) as { factory?: { key?: string } };

    expect(body.factory?.key).toBe("NEWWA");
  });

  it("updates a task title and description", async () => {
    const fixture = structuredClone(defaultFactoriesFixture);
    const response = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/orders/${OPEN_WORK_ORDER.id}`,
      {
        method: "PATCH",
        body: JSON.stringify({ title: "New title", description: "New body" }),
      },
      fixture,
    );
    const body = (await response.json()) as { order?: { title?: string; description?: string } };

    expect(body.order?.title).toBe("New title");
    expect(body.order?.description).toBe("New body");
  });

  it("does not serve a separate line-metrics route", async () => {
    const response = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/line-metrics`);
    expect(response.status).toBe(404);
  });

  it("lists and creates PR feedback handlers", async () => {
    const fixture = structuredClone(defaultFactoriesFixture);
    const list = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/pr-feedback-handlers`,
      undefined,
      fixture,
    );
    await expect(list.json()).resolves.toMatchObject({ handlers: [] });

    const created = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/pr-feedback-handlers`,
      {
        method: "POST",
        body: JSON.stringify({ source: "SOURCE_PULL_REQUEST_CHECKS", name: "Fix pull request checks" }),
      },
      fixture,
    );
    await expect(created.json()).resolves.toMatchObject({
      handler: {
        name: "Fix pull request checks",
        source: "SOURCE_PULL_REQUEST_CHECKS",
        healthy: true,
      },
    });

    const afterCreate = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/pr-feedback-handlers`,
      undefined,
      fixture,
    );
    const body = (await afterCreate.json()) as { handlers: Array<{ source?: string }> };
    expect(body.handlers).toHaveLength(1);
    expect(body.handlers[0]?.source).toBe("SOURCE_PULL_REQUEST_CHECKS");
  });

  it("applies Polar billing after a billing sync", async () => {
    const fixture = {
      ...structuredClone(defaultFactoriesFixture),
      billingSyncCalls: 0,
      billingAfterSync: BUSINESS_ORGANIZATION_BILLING,
    };

    const response = await fetchFactoryPageFixture(
      `/api/v1/organizations/${FACTORIES_ORGANIZATION_ID}/billing/sync`,
      { method: "POST", body: "{}" },
      fixture,
    );
    await expect(response.json()).resolves.toMatchObject({
      plan: "business",
      creditPurchaseAllowed: true,
    });
    expect(fixture.billingSyncCalls).toBe(1);
    expect(fixture.organizationBilling).toMatchObject({ plan: "business" });
  });

  it("cancels and resumes Polar Business on the billing routes", async () => {
    const fixture = {
      ...structuredClone(defaultFactoriesFixture),
      organizationBilling: { ...BUSINESS_ORGANIZATION_BILLING },
    };

    const canceled = await fetchFactoryPageFixture(
      `/api/v1/organizations/${FACTORIES_ORGANIZATION_ID}/billing/cancel`,
      { method: "POST", body: "{}" },
      fixture,
    );
    await expect(canceled.json()).resolves.toMatchObject({
      plan: "business",
      cancelAtPeriodEnd: true,
    });

    const resumed = await fetchFactoryPageFixture(
      `/api/v1/organizations/${FACTORIES_ORGANIZATION_ID}/billing/resume`,
      { method: "POST", body: "{}" },
      fixture,
    );
    await expect(resumed.json()).resolves.toMatchObject({
      plan: "business",
      cancelAtPeriodEnd: false,
    });
  });

  it("lists two pull requests per line-board column across draft, open, merged, and closed", async () => {
    const response = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/prs`,
      undefined,
      structuredClone(lineMetricsFactoriesFixture),
    );
    const body = (await response.json()) as {
      pullRequests: Array<{ workOrderId?: string; number?: string; state?: string }>;
    };
    const byOrder = Object.fromEntries(body.pullRequests.map((pullRequest) => [pullRequest.workOrderId, pullRequest]));

    expect(byOrder["wo-review-pay-842"]).toMatchObject({ number: "842", state: "STATE_DRAFT" });
    expect(byOrder["wo-review-pay-844"]).toMatchObject({ number: "844", state: "STATE_DRAFT" });
    expect(byOrder["wo-approval-refunds"]).toMatchObject({ number: "109", state: "STATE_OPEN" });
    expect(byOrder["wo-board-implement-notify"]).toMatchObject({ number: "114", state: "STATE_DRAFT" });
    expect(byOrder["wo-failed-refunds"]).toMatchObject({ number: "6812", state: "STATE_OPEN" });
    expect(byOrder["wo-open-refunds-schema"]).toMatchObject({ number: "102", state: "STATE_DRAFT" });
    expect(byOrder["wo-pr-closure-receipts"]).toMatchObject({ number: "510", state: "STATE_MERGED" });
    expect(byOrder["wo-board-done-rejected"]).toMatchObject({ number: "112", state: "STATE_CLOSED" });
  });

  it("returns a 7-day velocity series when periodDays is 7", async () => {
    const response = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/velocity?periodDays=7`);
    const body = (await response.json()) as { points?: unknown[] };

    expect(body.points).toHaveLength(7);
  });

  it("returns no planning session when the fixture does not seed one", async () => {
    const response = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/work-orders/${DRAFT_WORK_ORDER.id}/planning-session`,
    );

    expect(response.status).toBe(404);
  });

  it("serves a seeded planning session and stores a survey answer", async () => {
    const fixture = refineChatBoardFixture();
    const sessionPath = `/api/v1/factories/${PRIMARY_FACTORY_ID}/work-orders/${DRAFT_WORK_ORDER.id}/planning-session`;
    const loaded = await fetchFactoryPageFixture(sessionPath, undefined, fixture);
    const body = (await loaded.json()) as { session?: { id?: string; survey?: unknown } };

    expect(loaded.status).toBe(200);
    expect(body.session?.id).toBe("ps-draft-refunds");
    expect(body.session?.survey).toBeTruthy();

    const answered = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/planning-sessions/ps-draft-refunds/survey-answer`,
      {
        method: "POST",
        body: JSON.stringify({ text: "What should the first change include? Reactions on the task header only" }),
      },
      fixture,
    );
    const next = (await answered.json()) as {
      session?: { survey?: unknown; messages?: Array<{ text?: string }> };
    };

    expect(next.session?.survey).toBeNull();
    expect(next.session?.messages?.at(-1)?.text).toContain("Reactions on the task header only");
  });
});

describe("factory agent resources fixture", () => {
  it("lists MCP connections and creates a header connection", async () => {
    const fixture = {
      ...structuredClone(defaultFactoriesFixture),
      agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
    };

    const listed = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/agent-resources?kind=KIND_MCP_SERVER`,
      undefined,
      fixture,
    );
    await expect(listed.json()).resolves.toMatchObject({
      resources: [expect.objectContaining({ name: "docs", auth: "AUTH_HEADERS" })],
    });

    const created = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/agent-resources`,
      {
        method: "POST",
        body: JSON.stringify({
          kind: "KIND_MCP_SERVER",
          name: "mobbin",
          url: "https://api.mobbin.com/mcp",
          auth: "AUTH_OAUTH",
        }),
      },
      fixture,
    );
    await expect(created.json()).resolves.toMatchObject({
      resource: expect.objectContaining({
        name: "mobbin",
        auth: "AUTH_OAUTH",
        oauthStatus: "OAUTH_STATUS_NOT_CONNECTED",
      }),
    });
  });

  it("creates an inline skill", async () => {
    const fixture = structuredClone(defaultFactoriesFixture);

    const created = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/agent-resources`,
      {
        method: "POST",
        body: JSON.stringify({
          kind: "KIND_SKILL",
          name: "review-copy",
          markdown: "# Review copy",
        }),
      },
      fixture,
    );
    await expect(created.json()).resolves.toMatchObject({
      resource: expect.objectContaining({
        kind: "KIND_SKILL",
        name: "review-copy",
        markdown: "# Review copy",
      }),
    });
  });

  it("lists tools for an MCP server", async () => {
    const fixture = {
      ...structuredClone(defaultFactoriesFixture),
      agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [HEADER_MCP_RESOURCE] },
    };

    const listed = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/agent-resources/${HEADER_MCP_RESOURCE.id}/tools`,
      undefined,
      fixture,
    );
    await expect(listed.json()).resolves.toMatchObject({
      tools: expect.arrayContaining([expect.objectContaining({ name: "search" })]),
    });
  });
});
