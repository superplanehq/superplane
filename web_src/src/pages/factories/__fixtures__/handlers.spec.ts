import { describe, expect, it } from "vitest";

import { fetchFactoryPageFixture } from "./handlers";
import { CLOSED_WORK_ORDER, OPEN_WORK_ORDER, PRIMARY_FACTORY_ID, RUNNING_WORK_ORDER } from "./factoryPageResponses";

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
});
