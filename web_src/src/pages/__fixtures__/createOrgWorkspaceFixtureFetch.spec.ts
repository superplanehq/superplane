import { describe, expect, it, vi } from "vitest";

import { defaultFactoriesFixture, PRIMARY_FACTORY_ID } from "@/pages/factories/__fixtures__/factoryPageResponses";
import { SOFTWARE_FACTORY_APP_ID } from "@/pages/home/__fixtures__/homePageResponses";

import { createOrgWorkspaceFixtureFetch } from "./createOrgWorkspaceFixtureFetch";

function orgWorkspaceFetch(options?: Parameters<typeof createOrgWorkspaceFixtureFetch>[1]) {
  const fallback = vi.fn() as unknown as typeof fetch;
  return createOrgWorkspaceFixtureFetch(fallback, options);
}

describe("createOrgWorkspaceFixtureFetch", () => {
  it("prefers canvas-app integration definitions when appFixture is supplied", async () => {
    const fixtureFetch = orgWorkspaceFetch({
      appFixture: {
        organizationId: "org-1",
        canvasId: "canvas-1",
        integrations: {
          integrations: [{ name: "sentry", label: "Sentry", configuration: [] }],
        },
      },
    });

    const response = await fixtureFetch("http://localhost/api/v1/integrations");
    await expect(response.json()).resolves.toMatchObject({
      integrations: [expect.objectContaining({ name: "sentry" })],
    });
  });

  it("serves factory GitHub/Claude definitions when appFixture is omitted", async () => {
    const fixtureFetch = orgWorkspaceFetch();

    const response = await fixtureFetch("http://localhost/api/v1/integrations");
    const body = await response.json();
    expect(body.integrations.map((item: { name: string }) => item.name)).toEqual(["github", "claude"]);
  });

  it("keeps factory GitHub/Claude definitions when factoriesFixture is present without appFixture", async () => {
    const fixtureFetch = orgWorkspaceFetch({ factoriesFixture: defaultFactoriesFixture });

    const response = await fixtureFetch("http://localhost/api/v1/integrations");
    const body = await response.json();
    expect(body.integrations.map((item: { name: string }) => item.name)).toEqual(["github", "claude"]);
  });

  it("stamps factoryId on factory app canvases when appFixture is omitted", async () => {
    const fixtureFetch = orgWorkspaceFetch({ factoriesFixture: defaultFactoriesFixture });

    const response = await fixtureFetch("http://localhost/api/v1/canvases/app-refund-planner");
    await expect(response.json()).resolves.toMatchObject({
      canvas: {
        metadata: {
          id: "app-refund-planner",
          name: "Refund Planner",
          factoryId: PRIMARY_FACTORY_ID,
        },
      },
    });
  });

  it("leaves the home Software Factory canvas without factoryId", async () => {
    const fixtureFetch = orgWorkspaceFetch({ factoriesFixture: defaultFactoriesFixture });

    const response = await fixtureFetch(`http://localhost/api/v1/canvases/${SOFTWARE_FACTORY_APP_ID}`);
    const body = (await response.json()) as { canvas?: { metadata?: { factoryId?: string; id?: string } } };
    expect(body.canvas?.metadata?.factoryId).toBeUndefined();
    expect(body.canvas?.metadata?.id).not.toBe("app-refund-planner");
  });
});
