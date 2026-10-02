import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { CustomProviderConnectDialog } from "./CustomProviderConnectDialog";

const PROVIDER_ERROR = "The provider rejected this token.";

const mocks = vi.hoisted(() => ({
  createIntegration: vi.fn(),
  deleteIntegration: vi.fn(),
  showErrorToast: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", () => ({
  organizationsDeleteIntegration: mocks.deleteIntegration,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  integrationKeys: {
    connected: (organizationId: string) => ["integrations", "connected", organizationId],
    integration: (organizationId: string, integrationId: string) => [
      "integrations",
      "connected",
      organizationId,
      integrationId,
    ],
  },
  useCreateIntegration: () => ({ mutateAsync: mocks.createIntegration, isPending: false }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: mocks.showErrorToast,
}));

function renderDialog() {
  const onSelectionsChange = vi.fn();
  const onClose = vi.fn();
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <CustomProviderConnectDialog
        open
        organizationId="org-1"
        existingNames={new Set()}
        selections={{}}
        onSelectionsChange={onSelectionsChange}
        onClose={onClose}
      />
    </QueryClientProvider>,
  );
  return { onSelectionsChange, onClose };
}

async function submitConnection(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByTestId("onboarding-custom-url"), "https://models.example/v1");
  await user.type(screen.getByTestId("onboarding-custom-token"), "secret-token");
  await user.click(screen.getByTestId("onboarding-custom-api-type"));
  await user.click(screen.getByRole("option", { name: "OpenAI-compatible" }));
  await user.click(screen.getByRole("button", { name: "Connect" }));
}

describe("CustomProviderConnectDialog", () => {
  beforeEach(() => {
    mocks.createIntegration.mockReset();
    mocks.deleteIntegration.mockReset();
    mocks.deleteIntegration.mockResolvedValue({});
    mocks.showErrorToast.mockReset();
  });

  it("removes a connection that does not become ready", async () => {
    mocks.createIntegration.mockResolvedValue({
      data: {
        integration: {
          metadata: { id: "integration-bad", name: "customLlm" },
          status: { state: "error", stateDescription: PROVIDER_ERROR },
        },
      },
    });
    const user = userEvent.setup();
    const { onSelectionsChange, onClose } = renderDialog();

    await submitConnection(user);

    expect(mocks.deleteIntegration).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { id: "org-1", integrationId: "integration-bad" },
      }),
    );
    expect(mocks.showErrorToast).toHaveBeenCalledWith(PROVIDER_ERROR);
    expect(onSelectionsChange).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByRole("heading", { name: "Custom provider" })).toBeInTheDocument();
  });

  it("keeps a ready connection selected", async () => {
    mocks.createIntegration.mockResolvedValue({
      data: {
        integration: {
          metadata: { id: "integration-ok", name: "customLlm" },
          status: { state: "ready" },
        },
      },
    });
    const user = userEvent.setup();
    const { onSelectionsChange } = renderDialog();

    await submitConnection(user);

    expect(mocks.deleteIntegration).not.toHaveBeenCalled();
    expect(onSelectionsChange).toHaveBeenCalledWith({
      customLlm: { id: "integration-ok", name: "customLlm", ready: true },
    });
  });
});
