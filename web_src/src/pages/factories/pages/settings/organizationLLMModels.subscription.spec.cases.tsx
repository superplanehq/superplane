import { QueryClient } from "@tanstack/react-query";
import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, type vi } from "bun:test";
import type { FactoriesFactoryCodexConnection } from "@/api-client";
import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { factoryCodexConnectionKey } from "../onboarding/useFactoryCodexConnection";
import { renderPage } from "./organizationLLMModels.spec.render";

export function subscriptionModelSourceSwitchCases({
  setConnection,
  switchSource,
}: {
  setConnection: (connection: FactoriesFactoryCodexConnection) => void;
  switchSource: ReturnType<typeof vi.fn>;
}) {
  it("connects the admin before switching to a Codex subscription", async () => {
    setConnection({ status: "disconnected", eligible: true });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderPage("AGENT_HARNESS_SUPERPLANE", false, client);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Use Codex subscription" }));
    await user.click(screen.getByRole("button", { name: "Connect Codex" }));
    await user.click(screen.getByRole("button", { name: "Connect ChatGPT" }));
    expect(switchSource).not.toHaveBeenCalled();
    const codexConnection = { status: "ready", eligible: true, availableModels: ["gpt-5.4"] };
    setConnection(codexConnection);
    await act(async () => {
      client.setQueryData(factoryCodexConnectionKey("org-1", REFUND_FACTORY.id ?? "factory-1"), codexConnection);
    });
    await waitFor(() => expect(switchSource).toHaveBeenCalledTimes(1));
    expect(switchSource).toHaveBeenCalledWith({ source: "subscription", apiKey: undefined });
  });

  it("keeps a failed subscription switch visible and retryable", async () => {
    setConnection({ status: "ready", eligible: true, availableModels: ["gpt-5.4"] });
    switchSource.mockRejectedValueOnce(new Error("The intake configuration changed."));
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Use Codex subscription" }));
    await user.click(screen.getByRole("button", { name: "Connect Codex" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not switch the model source. Try again.");
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));
    await waitFor(() => expect(switchSource).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("switches a subscription workspace back to hosted models", async () => {
    setConnection({ status: "ready", eligible: true, availableModels: ["gpt-5.4"] });
    renderPage("AGENT_HARNESS_CODEX", true);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Use SuperPlane" }));
    await user.click(screen.getByRole("button", { name: "Switch to SuperPlane" }));
    expect(switchSource).toHaveBeenCalledWith({ source: "hosted", apiKey: undefined });
  });
}
