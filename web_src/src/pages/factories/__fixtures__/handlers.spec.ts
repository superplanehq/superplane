import { describe, expect, it } from "vitest";

import { fetchFactoryPageFixture } from "./handlers";
import { lineMetricsFactoriesFixture } from "./lineMetricsFactoriesFixture";
import {
  CLOSED_WORK_ORDER,
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
      factories: expect.arrayContaining([expect.objectContaining({ name: "Refunds Factory" })]),
    });

    const detail = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}`);
    const body = (await detail.json()) as {
      factory: { id?: string; name?: string; lines?: Array<{ id?: string; metrics?: { successRatePct?: number } }> };
    };
    expect(body.factory).toMatchObject({ id: PRIMARY_FACTORY_ID, name: "Refunds Factory" });
    const plan = body.factory.lines?.find((line) => line.id === REFUND_LINE_PLAN_ID);
    expect(plan?.metrics?.successRatePct).toBe(82);
  });

  it("serves work orders and includes both open and closed entries", async () => {
    const orders = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/orders`);
    const body = (await orders.json()) as { orders: Array<{ id?: string; state?: string }> };
    const ids = body.orders.map((entry) => entry.id);
    expect(ids).toEqual(expect.arrayContaining([OPEN_WORK_ORDER.id, RUNNING_WORK_ORDER.id, CLOSED_WORK_ORDER.id]));
  });

  it("returns factory apps for the populated factory", async () => {
    const apps = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/apps`);
    await expect(apps.json()).resolves.toMatchObject({
      apps: expect.arrayContaining([expect.objectContaining({ name: "Refund Planner" })]),
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

  it("does not serve a separate line-metrics route", async () => {
    const response = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/line-metrics`);
    expect(response.status).toBe(404);
  });
});
