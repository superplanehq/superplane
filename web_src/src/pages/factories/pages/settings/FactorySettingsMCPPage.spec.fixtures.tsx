import { render } from "@testing-library/react";

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

export function connectedMcpClient(id: string, clientName: string): FactoriesFactoryMcpClient {
  return {
    id,
    clientName,
    userId: STORYBOOK_ME_USER_ID,
    userName: STORYBOOK_ME_USER_NAME,
    userAvatarUrl: "/storybook/leonardo-dicaprio.jpg",
    createdAt: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
  };
}

export function renderSuperplaneMcpServer(clients?: FactoriesFactoryMcpClient[]) {
  render(
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/superplane-mcp-server`}
      factoriesFixture={
        clients
          ? { ...defaultFactoriesFixture, mcpClientsByFactoryId: { [PRIMARY_FACTORY_ID]: clients } }
          : defaultFactoriesFixture
      }
      experimentalFeatures={[FEATURE_SUPERPLANE_MCP_SERVER]}
    />,
  );
}
