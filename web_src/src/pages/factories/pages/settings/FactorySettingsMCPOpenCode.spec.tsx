import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FEATURE_SUPERPLANE_MCP_SERVER } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "../../__fixtures__/factoryPageResponses";

const superplaneMcpServerPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/superplane-mcp-server`;

describe("Factory MCP OpenCode token", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
    Element.prototype.scrollIntoView ??= vi.fn();
  });

  it("copies a file reference and shows the secret one time", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={superplaneMcpServerPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={[FEATURE_SUPERPLANE_MCP_SERVER]}
      />,
    );

    expect(await screen.findByTestId("superplane-mcp-clients-empty", {}, { timeout: 8000 })).toBeInTheDocument();
    await user.click(screen.getByTestId("superplane-mcp-connect-client"));
    await user.click(screen.getByTestId("superplane-mcp-client-tool-opencode"));
    expect(screen.getByText("Name the token so you can revoke it later.")).toBeInTheDocument();
    expect(screen.getByText("Paste this configuration. Replace the file path.")).toBeInTheDocument();

    const writeText = vi.fn(() => Promise.resolve());
    const previousClipboard = navigator.clipboard;
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    try {
      await user.click(screen.getByTestId("superplane-mcp-config-copy-opencode"));
      expect(writeText).toHaveBeenCalled();
      const copied = String(writeText.mock.calls[0]?.[0]);
      const parsed = JSON.parse(copied) as {
        mcp?: { superplane?: { type?: string; url?: string } };
        servers?: unknown;
      };
      expect(parsed.servers).toBeUndefined();
      expect(parsed.mcp?.superplane?.type).toBe("remote");
      expect(parsed.mcp?.superplane?.url).toBe(`${window.location.origin}/mcp`);
      expect(copied).toContain("{file:/path/to/superplane.key}");
      expect(copied).toContain("oauth");
      expect(copied).not.toContain("sp_mcp_");
    } finally {
      Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: previousClipboard,
      });
    }

    await user.type(screen.getByTestId("superplane-mcp-opencode-name"), "Build server");
    await user.click(screen.getByTestId("superplane-mcp-opencode-create"));
    expect(await screen.findByTestId("superplane-mcp-opencode-secret")).toHaveTextContent("sp_mcp_secret-shown-once");
    await user.click(screen.getByTestId("superplane-mcp-opencode-secret-done"));
    expect(screen.queryByTestId("superplane-mcp-opencode-secret")).not.toBeInTheDocument();
  }, 10000);

  it("finds a token by name and says revoke stops access at once", async () => {
    const user = userEvent.setup();
    const tokenId = "mcp-api-token-build";
    render(
      <FactoriesHarness
        pathSuffix={superplaneMcpServerPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          mcpClientsByFactoryId: {
            [PRIMARY_FACTORY_ID]: [
              {
                id: tokenId,
                kind: "api_token",
                clientName: "Build server",
                userId: STORYBOOK_ME_USER_ID,
                userName: STORYBOOK_ME_USER_NAME,
                createdAt: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
              },
            ],
          },
        }}
        experimentalFeatures={[FEATURE_SUPERPLANE_MCP_SERVER]}
      />,
    );

    expect(await screen.findByTestId("superplane-mcp-clients-list", {}, { timeout: 8000 })).toHaveTextContent(
      "Build server",
    );
    await user.type(screen.getByTestId("superplane-mcp-clients-search"), "Build server");
    expect(screen.getByTestId(`superplane-mcp-client-${tokenId}`)).toBeInTheDocument();
    await user.click(screen.getByTestId(`superplane-mcp-client-revoke-${tokenId}`));
    expect(screen.getByText("This stops access at once. You cannot undo this action.")).toBeInTheDocument();
    await user.click(screen.getByTestId("factory-delete-confirm-button"));
    await waitFor(() => {
      expect(screen.queryByTestId("superplane-mcp-clients-list")).not.toBeInTheDocument();
    });
  }, 10000);
});
