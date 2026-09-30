import { describe, expect, it } from "bun:test";

import { fetchFactoryPageFixture } from "./handlers";
import { defaultFactoriesFixture, PRIMARY_FACTORY_ID } from "./factoryPageResponses";

describe("factory SuperPlane MCP clients fixture", () => {
  it("lists and revokes a connected client", async () => {
    const fixture = {
      ...structuredClone(defaultFactoriesFixture),
      mcpClientsByFactoryId: {
        [PRIMARY_FACTORY_ID]: [{ id: "mcp-client-1", clientName: "Cursor", userName: "Ada" }],
      },
    };

    const listed = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/mcp-clients`,
      undefined,
      fixture,
    );
    await expect(listed.json()).resolves.toMatchObject({
      clients: [expect.objectContaining({ id: "mcp-client-1", clientName: "Cursor" })],
    });

    await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/mcp-clients/mcp-client-1`,
      { method: "DELETE" },
      fixture,
    );
    const after = await fetchFactoryPageFixture(
      `/api/v1/factories/${PRIMARY_FACTORY_ID}/mcp-clients`,
      undefined,
      fixture,
    );
    await expect(after.json()).resolves.toEqual({ clients: [] });
  });
});
