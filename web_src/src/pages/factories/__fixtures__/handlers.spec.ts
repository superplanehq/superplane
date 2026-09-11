import { describe, expect, it } from "vitest";

import { fetchFactoryPageFixture } from "./handlers";
import { lineMetricsFactoriesFixture } from "./lineMetricsFactoriesFixture";
import {
  CLOSED_WORK_ORDER,
  defaultFactoriesFixture,
  FACTORIES_ORGANIZATION_ID,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  REFUND_LINE_ONBOARDING_ID,
  REFUND_LINE_PLAN_ID,
  RUNNING_WORK_ORDER,
} from "./factoryPageResponses";
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

  it("returns factory apps for the populated factory", async () => {
    const apps = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/apps`);
    await expect(apps.json()).resolves.toMatchObject({
      apps: expect.arrayContaining([expect.objectContaining({ name: "Refund Planner" })]),
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
});
