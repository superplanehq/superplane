import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { IntegrationsIntegrationDefinition, OrganizationsCreateIntegrationResponse } from "@/api-client";
import { IntegrationCreateDialog } from "./index";

// Minimal stand-in for the Claude integration definition: a required, sensitive
// "apiKey" field. This is the field that triggers the reported bug when left empty.
function claudeLikeDefinition(): IntegrationsIntegrationDefinition {
  return {
    name: "claude",
    icon: "loader",
    configuration: [
      {
        name: "apiKey",
        label: "API Key",
        type: "string",
        sensitive: true,
        required: true,
      },
    ],
  } as IntegrationsIntegrationDefinition;
}

function renderDialog(onCreateIntegration: (payload: unknown) => Promise<OrganizationsCreateIntegrationResponse>) {
  const queryClient = new QueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <IntegrationCreateDialog
        open={true}
        onOpenChange={() => {}}
        integrationDefinition={claudeLikeDefinition()}
        organizationId="org-123"
        onCreateIntegration={
          onCreateIntegration as unknown as (payload: {
            integrationName: string;
            name: string;
            configuration?: Record<string, unknown>;
          }) => Promise<OrganizationsCreateIntegrationResponse>
        }
      />
    </QueryClientProvider>,
  );
}

describe("IntegrationCreateDialog", () => {
  it("keeps Connect disabled while the required API key is empty", async () => {
    const onCreateIntegration = vi.fn();
    renderDialog(onCreateIntegration);

    const user = userEvent.setup();
    await user.type(screen.getByPlaceholderText("e.g., my-app-integration"), "my-claude");

    const connectButton = screen.getByRole("button", { name: "Connect" });
    expect(connectButton).toBeDisabled();

    await user.click(connectButton);
    expect(onCreateIntegration).not.toHaveBeenCalled();
  });

  it("enables Connect once the API key has a value, and calls onCreateIntegration", async () => {
    const onCreateIntegration = vi.fn().mockResolvedValue({ integration: { metadata: { id: "int-1" } } });
    renderDialog(onCreateIntegration);

    const user = userEvent.setup();
    await user.type(screen.getByPlaceholderText("e.g., my-app-integration"), "my-claude");

    const apiKeyInput = document.querySelector('input[type="password"]');
    expect(apiKeyInput).not.toBeNull();
    await user.type(apiKeyInput as Element, "sk-secret");

    const connectButton = screen.getByRole("button", { name: "Connect" });
    await waitFor(() => expect(connectButton).toBeEnabled());

    await user.click(connectButton);
    await waitFor(() => expect(onCreateIntegration).toHaveBeenCalledTimes(1));
    expect(onCreateIntegration).toHaveBeenCalledWith(
      expect.objectContaining({
        integrationName: "claude",
        name: "my-claude",
        configuration: expect.objectContaining({ apiKey: "sk-secret" }),
      }),
    );
  });
});
