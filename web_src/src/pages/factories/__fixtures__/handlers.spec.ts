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

  it("serves workspace settings members, roles, secrets, and invite link", async () => {
    const users = await fetchFactoryPageFixture("/api/v1/users");
    await expect(users.json()).resolves.toMatchObject({
      users: expect.arrayContaining([
        expect.objectContaining({
          spec: { displayName: "Storybook User" },
          status: { roles: [expect.objectContaining({ roleName: "org_owner", roleDisplayName: "Owner" })] },
        }),
      ]),
    });

    const roles = await fetchFactoryPageFixture("/api/v1/roles");
    await expect(roles.json()).resolves.toMatchObject({
      roles: expect.arrayContaining([expect.objectContaining({ metadata: { name: "org_admin" } })]),
    });

    const secrets = await fetchFactoryPageFixture("/api/v1/secrets");
    await expect(secrets.json()).resolves.toMatchObject({
      secrets: expect.arrayContaining([
        expect.objectContaining({ metadata: expect.objectContaining({ name: "openai" }) }),
      ]),
    });

    const secret = await fetchFactoryPageFixture("/api/v1/secrets/secret-openai");
    await expect(secret.json()).resolves.toMatchObject({
      secret: expect.objectContaining({ metadata: { id: "secret-openai", name: "openai" } }),
    });

    const invite = await fetchFactoryPageFixture("/api/v1/organizations/org-1/invite-link");
    await expect(invite.json()).resolves.toMatchObject({
      inviteLink: expect.objectContaining({ token: "storybook-invite-token", enabled: true }),
    });
  });

  it("serves the organization integrations catalog and connected instances", async () => {
    const catalog = await fetchFactoryPageFixture("/api/v1/integrations");
    const catalogBody = (await catalog.json()) as { integrations: Array<{ name?: string; description?: string }> };
    const names = catalogBody.integrations.map((entry) => entry.name);
    expect(names).toEqual(
      expect.arrayContaining(["github", "gitlab", "slack", "linear", "jira", "claude", "datadog", "aws"]),
    );
    expect(catalogBody.integrations.length).toBeGreaterThan(20);
    expect(catalogBody.integrations.find((entry) => entry.name === "github")?.description).toBe(
      "Manage and react to changes in your GitHub repositories",
    );

    const connected = await fetchFactoryPageFixture("/api/v1/organizations/org-1/integrations");
    await expect(connected.json()).resolves.toMatchObject({
      integrations: expect.arrayContaining([
        expect.objectContaining({
          metadata: expect.objectContaining({ integrationName: "github" }),
          status: expect.objectContaining({ state: "ready" }),
        }),
      ]),
    });
  });
});
