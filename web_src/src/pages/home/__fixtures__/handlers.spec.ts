import { describe, expect, it, vi } from "vitest";

import { createHomeFixtureFetch } from "./handlers";

async function fetchFixture(path: string): Promise<Response> {
  const fallback = vi.fn() as unknown as typeof fetch;
  const fixtureFetch = createHomeFixtureFetch(fallback);
  return fixtureFetch(`http://localhost${path}`);
}

describe("createHomeFixtureFetch", () => {
  it("serves populated canvases and folders", async () => {
    const canvases = await fetchFixture("/api/v1/canvases");
    await expect(canvases.json()).resolves.toMatchObject({
      canvases: expect.arrayContaining([expect.objectContaining({ name: "Software Factory" })]),
    });

    const folders = await fetchFixture("/api/v1/canvas-folders");
    await expect(folders.json()).resolves.toMatchObject({
      folders: expect.arrayContaining([
        expect.objectContaining({ spec: expect.objectContaining({ title: "Automation" }) }),
      ]),
    });
  });

  it("serves account and me for providers", async () => {
    const account = await fetchFixture("/account");
    await expect(account.json()).resolves.toMatchObject({
      id: "storybook-user",
      name: "Storybook User",
    });

    const me = await fetchFixture("/api/v1/me");
    const meBody = await me.json();
    expect(meBody).toMatchObject({
      user: expect.objectContaining({ organizationId: expect.any(String) }),
    });
    expect(meBody.user.permissions).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ resource: "agents", action: "read" }),
        expect.objectContaining({ resource: "agents", action: "create" }),
      ]),
    );
  });

  it("exposes the managed-agents experimental feature", async () => {
    const features = await fetchFixture("/account/experimental-features");
    await expect(features.json()).resolves.toMatchObject({
      features: expect.arrayContaining([expect.objectContaining({ id: "claude_managed_agents", released: true })]),
    });
  });
});
