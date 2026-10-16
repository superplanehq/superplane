import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { unmockedSrc } from "@/test/unmockedModule";
import { IntegrationCreateDialog } from ".";

vi.mock("@/hooks/useIntegrations", () => ({
  ...unmockedSrc<Record<string, unknown>>("hooks/useIntegrations"),
  useUpdateIntegration: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

vi.mock("@/lib/toast", () => ({ showErrorToast: vi.fn(), showSuccessToast: vi.fn() }));

describe("IntegrationCreateDialog", () => {
  it("shows key validation errors and lets the user correct the key", async () => {
    const user = userEvent.setup();
    const message = "Unable to verify the API key. Check the key and try again.";
    const create = vi
      .fn()
      .mockRejectedValueOnce({ message })
      .mockResolvedValueOnce({
        integration: { metadata: { id: "key-1", name: "my-key" }, status: { state: "ready" } },
      });
    const onCreated = vi.fn();
    const onOpenChange = vi.fn();
    render(
      <QueryClientProvider client={new QueryClient()}>
        <IntegrationCreateDialog
          open
          organizationId="org-1"
          defaultName="my-key"
          integrationDefinition={{
            name: "claude",
            label: "Anthropic",
            configuration: [{ name: "apiKey", label: "API key", type: "string", required: true, sensitive: true }],
          }}
          onCreateIntegration={create}
          onCreated={onCreated}
          onOpenChange={onOpenChange}
        />
      </QueryClientProvider>,
    );
    const key = screen.getByTestId("string-field-apikey");
    await user.type(key, "invalid-key");
    await user.click(screen.getByRole("button", { name: "Connect" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(message);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(onCreated).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);

    await user.clear(key);
    await user.type(key, "corrected-key");
    await user.click(screen.getByRole("button", { name: "Connect" }));

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith("key-1", "my-key"));
    expect(create.mock.calls[1][0].configuration.apiKey).toBe("corrected-key");
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
