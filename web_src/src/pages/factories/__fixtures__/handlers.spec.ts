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

  it("serves factory usage and organization LLM spend reports", async () => {
    const usage = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/usage`);
    await expect(usage.json()).resolves.toMatchObject({
      totalTokens: "25600",
      totalCostCents: "876",
      byModel: expect.arrayContaining([expect.objectContaining({ provider: "anthropic" })]),
    });

    const spend = await fetchFactoryPageFixture(`/api/v1/organizations/${FACTORIES_ORGANIZATION_ID}/llm-spend`);
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
});
