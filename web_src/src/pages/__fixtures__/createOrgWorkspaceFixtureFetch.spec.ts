import { describe, expect, it, vi } from "vitest";

import { createOrgWorkspaceFixtureFetch } from "./createOrgWorkspaceFixtureFetch";

describe("createOrgWorkspaceFixtureFetch", () => {
  it("prefers canvas-app integration definitions when appFixture is supplied", async () => {
    const fallback = vi.fn() as unknown as typeof fetch;
    const fixtureFetch = createOrgWorkspaceFixtureFetch(fallback, {
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
    const fallback = vi.fn() as unknown as typeof fetch;
    const fixtureFetch = createOrgWorkspaceFixtureFetch(fallback);

    const response = await fixtureFetch("http://localhost/api/v1/integrations");
    const body = await response.json();
    expect(body.integrations.map((item: { name: string }) => item.name)).toEqual(["github", "claude"]);
  });
});
