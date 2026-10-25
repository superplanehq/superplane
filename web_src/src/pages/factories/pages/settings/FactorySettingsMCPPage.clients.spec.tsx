import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { STORYBOOK_ME_USER_NAME } from "../../__fixtures__/factoryPageResponses";
import { connectedMcpClient, renderSuperplaneMcpServer } from "./FactorySettingsMCPPage.spec.fixtures";

describe("FactorySettingsMCPPage connected clients", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
    Element.prototype.scrollIntoView ??= vi.fn();
  });

  it("lists a connected SuperPlane MCP client and revokes it", async () => {
    const user = userEvent.setup();
    const clientId = "mcp-client-cursor";
    renderSuperplaneMcpServer([connectedMcpClient(clientId, "Cursor")]);

    expect(await screen.findByTestId("superplane-mcp-clients-list", {}, { timeout: 8000 })).toHaveTextContent("Cursor");
    expect(screen.getByTestId(`superplane-mcp-client-${clientId}`)).toHaveTextContent(STORYBOOK_ME_USER_NAME);

    await user.click(screen.getByTestId(`superplane-mcp-client-revoke-${clientId}`));
    expect(screen.getByText(`Revoke "Cursor"?`)).toBeInTheDocument();
    await user.click(screen.getByTestId("factory-delete-confirm-button"));
    await waitFor(() => {
      expect(screen.queryByTestId("superplane-mcp-clients-list")).not.toBeInTheDocument();
    });
    expect(await screen.findByTestId("superplane-mcp-clients-empty")).toBeInTheDocument();
  }, 10000);

  it("warns that revoke ends every workspace on a multi-workspace connection", async () => {
    const user = userEvent.setup();
    const clientId = "mcp-client-shared";
    renderSuperplaneMcpServer([{ ...connectedMcpClient(clientId, "Cursor"), workspaceCount: 2 }]);

    await user.click(await screen.findByTestId(`superplane-mcp-client-revoke-${clientId}`, {}, { timeout: 8000 }));
    expect(
      screen.getByText(
        "This disconnects the client from every workspace on this connection. The client must sign in again.",
      ),
    ).toBeInTheDocument();
  }, 10000);

  it("shows the full MCP client name on hover and in the revoke dialog", async () => {
    const longClientId = "mcp-client-long-name";
    const longClientName = "https://example.com/oauth/clients/workspace-example/client-metadata.json";
    renderSuperplaneMcpServer([
      connectedMcpClient("mcp-client-cursor", "Cursor"),
      connectedMcpClient(longClientId, longClientName),
    ]);

    const table = await screen.findByRole("table", { name: "Connected clients" }, { timeout: 8000 });
    const longRow = within(table).getByRole("row", { name: (name) => name.includes(longClientName) });
    expect(within(longRow).getAllByRole("cell")).toHaveLength(4);
    expect(within(table).getByRole("columnheader", { name: "Client" })).toBeInTheDocument();
    const longName = within(longRow).getByTestId(`superplane-mcp-client-name-${longClientId}`);
    expect(screen.getByTestId("superplane-mcp-client-mcp-client-cursor")).toHaveTextContent("Cursor");
    expect(longName).toHaveAttribute("title", longClientName);
    expect(longName).toHaveTextContent(longClientName);
    expect(within(longRow).getByRole("button", { name: "Revoke" })).toBeEnabled();

    await userEvent.setup().click(within(longRow).getByRole("button", { name: "Revoke" }));
    expect(screen.getByText(`Revoke "${longClientName}"?`)).toBeInTheDocument();
  }, 10000);
});
