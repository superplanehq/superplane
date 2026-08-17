import { describe, expect, it } from "vitest";

import { fetchFactoryPageFixture } from "./handlers";
import {
  CLOSED_WORK_ORDER,
  EMPTY_FACTORY_ID,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  REFUND_LINE_PLAN_METRICS,
  RUNNING_WORK_ORDER,
} from "./factoryPageResponses";

describe("matchFactoryPageFixture", () => {
  it("lists factories and returns the primary factory by id", async () => {
    const list = await fetchFactoryPageFixture("/api/v1/factories");
    await expect(list.json()).resolves.toMatchObject({
      factories: expect.arrayContaining([expect.objectContaining({ name: "Refunds Factory" })]),
    });

    const detail = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}`);
    await expect(detail.json()).resolves.toMatchObject({
      factory: expect.objectContaining({ id: PRIMARY_FACTORY_ID, name: "Refunds Factory" }),
    });
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

  it("serves line metrics ahead of the generic lines/:id route", async () => {
    const response = await fetchFactoryPageFixture(`/api/v1/factories/${PRIMARY_FACTORY_ID}/lines/metrics`);
    await expect(response.json()).resolves.toEqual({ metrics: [REFUND_LINE_PLAN_METRICS] });
  });

  it("returns an empty metrics list for a factory with no line metrics", async () => {
    const response = await fetchFactoryPageFixture(`/api/v1/factories/${EMPTY_FACTORY_ID}/lines/metrics`);
    await expect(response.json()).resolves.toEqual({ metrics: [] });
  });
});
