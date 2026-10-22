import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "bun:test";

import { client } from "@/api-client/client.gen";
import type { FactoriesFactoryMcpClient } from "@/api-client";
import { FEATURE_SUPERPLANE_MCP_SERVER } from "@/lib/experimentalFeatures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageResponses";

function mcpClient(id: string, clientName: string, otherWorkspaceNames: string[]): FactoriesFactoryMcpClient {
  return {
    id,
    clientName,
    otherWorkspaceNames,
    userId: STORYBOOK_ME_USER_ID,
    userName: STORYBOOK_ME_USER_NAME,
    createdAt: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
  };
}

describe("SuperPlaneMCPServerSection", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
  });

  it("names the other workspaces a connection reaches and limits revoke to this workspace", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/superplane-mcp-server`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          mcpClientsByFactoryId: {
            [PRIMARY_FACTORY_ID]: [mcpClient("shared", "Cursor", ["Beta", "Gamma"]), mcpClient("single", "Codex", [])],
          },
        }}
        experimentalFeatures={[FEATURE_SUPERPLANE_MCP_SERVER]}
      />,
    );

    const shared = await screen.findByTestId("superplane-mcp-client-shared", {}, { timeout: 8000 });
    expect(shared).toHaveTextContent("Also reaches: Beta, Gamma");
    expect(screen.getByTestId("superplane-mcp-client-single")).not.toHaveTextContent("Also reaches");

    await user.click(screen.getByTestId("superplane-mcp-client-revoke-shared"));
    expect(screen.getByText(/access to this workspace only/)).toBeInTheDocument();
  }, 10000);
});
