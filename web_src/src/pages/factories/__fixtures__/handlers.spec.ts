import { describe, expect, it } from "vitest";

import { fetchFactoryPageFixture } from "./handlers";
import {
  CLOSED_WORK_ORDER,
  defaultFactoriesFixture,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
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

  it("adds and removes a work order reaction", async () => {
    // Share one cloned fixture across both calls so the removal sees the
    // reaction the addition just created (each call would otherwise get
    // its own fresh clone of `defaultFactoriesFixture`).
    const fixture = structuredClone(defaultFactoriesFixture);

    const added = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/orders/${OPEN_WORK_ORDER.id}/reactions`,
      { method: "POST", body: JSON.stringify({ content: "rocket" }) },
      fixture,
    );
    await expect(added.json()).resolves.toMatchObject({
      reactions: expect.arrayContaining([expect.objectContaining({ content: "rocket", count: 1, reactedByMe: true })]),
    });

    const removed = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/orders/${OPEN_WORK_ORDER.id}/reactions/rocket`,
      { method: "DELETE" },
      fixture,
    );
    const removedBody = (await removed.json()) as { reactions: Array<{ content?: string }> };
    expect(removedBody.reactions.some((reaction) => reaction.content === "rocket")).toBe(false);
  });
});
