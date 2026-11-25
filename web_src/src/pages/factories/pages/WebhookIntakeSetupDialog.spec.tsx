import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";

import { WebhookIntakeSetupDialog } from "./WebhookIntakeSetupDialog";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
  createWebhook: vi.fn(),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
}));

vi.mock("@/hooks/useFactoryWebhooks", () => ({
  useFactoryWebhookMutations: () => ({ create: { mutateAsync: mocks.createWebhook, isPending: false } }),
  useFactoryWebhooks: () => ({
    data: [{ id: "webhook-existing", name: "Tracker bridge" }],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

function renderDialog(onCreated = vi.fn()) {
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <TooltipProvider>
          <WebhookIntakeSetupDialog
            organizationId="org-1"
            factoryId="factory-1"
            factoryKey="sp"
            onClose={vi.fn()}
            onCreated={onCreated}
          />
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return onCreated;
}

describe("WebhookIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ intake: { id: "intake-1" } });
    mocks.createWebhook.mockReset();
    mocks.createWebhook.mockResolvedValue({
      webhook: { id: "webhook-new", name: "Webhook intake", url: "https://hooks.example.com/webhook-new" },
      secret: "secret-value",
    });
  });

  it("creates a webhook, then shows its secret before the dialog closes", async () => {
    const user = userEvent.setup();
    const onCreated = renderDialog();

    await user.click(screen.getByTestId("webhook-intake-setup-continue"));
    await user.clear(screen.getByTestId("webhook-intake-setup-title-path"));
    await user.type(screen.getByTestId("webhook-intake-setup-title-path"), "issue.title");
    await user.click(screen.getByTestId("webhook-intake-setup-finish"));

    expect(await screen.findByTestId("factory-webhook-secret")).toHaveTextContent("secret-value");
    expect(mocks.createIntake).toHaveBeenCalledWith({
      source: "SOURCE_WEBHOOK",
      resourceId: "webhook-new",
      settings: { webhookTitlePath: "issue.title", webhookDescriptionPath: "description", webhookLinkPath: "url" },
    });
    expect(onCreated).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("factory-webhook-secret-done"));
    expect(onCreated).toHaveBeenCalled();
  });

  it("reuses the new webhook when the intake create fails and the user tries again", async () => {
    const user = userEvent.setup();
    renderDialog();
    mocks.createIntake.mockRejectedValueOnce(new Error("boom"));

    await user.click(screen.getByTestId("webhook-intake-setup-continue"));
    await user.click(screen.getByTestId("webhook-intake-setup-finish"));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    await user.click(screen.getByTestId("webhook-intake-setup-finish"));

    await screen.findByTestId("factory-webhook-secret");
    expect(mocks.createWebhook).toHaveBeenCalledTimes(1);
  });

  it("creates the intake for an existing webhook without a secret step", async () => {
    const user = userEvent.setup();
    const onCreated = renderDialog();

    await user.click(screen.getByTestId("webhook-intake-mode-existing"));
    expect(screen.getByTestId("webhook-intake-setup-continue")).toBeDisabled();
    await user.click(screen.getByTestId("custom-automation-webhook-select"));
    await user.click(await screen.findByRole("option", { name: "Tracker bridge" }));
    await user.click(screen.getByTestId("webhook-intake-setup-continue"));
    await user.click(screen.getByTestId("webhook-intake-setup-finish"));

    await waitFor(() => expect(onCreated).toHaveBeenCalled());
    expect(mocks.createWebhook).not.toHaveBeenCalled();
    expect(mocks.createIntake).toHaveBeenCalledWith(expect.objectContaining({ resourceId: "webhook-existing" }));
    expect(screen.queryByTestId("factory-webhook-secret-dialog")).not.toBeInTheDocument();
  });

  it("blocks finish while the title path is empty", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("webhook-intake-setup-continue"));
    await user.clear(screen.getByTestId("webhook-intake-setup-title-path"));

    expect(screen.getByTestId("webhook-intake-setup-finish")).toBeDisabled();
  });
});
