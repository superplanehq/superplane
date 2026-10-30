import { QueryClient } from "@tanstack/react-query";
import { act, screen, waitFor, within } from "@testing-library/react";
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
  it("returns from sign-in to owner selection and requires explicit confirmation", async () => {
    setConnection({ status: "disconnected", eligible: true });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderPage("AGENT_HARNESS_SUPERPLANE", false, client);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("tab", { name: "Your ChatGPT subscription" }));
    expect(screen.getByRole("button", { name: "Switch to Codex" })).toBeDisabled();
    await user.click(await screen.findByRole("button", { name: "Connect Codex" }));
    await user.click(await screen.findByRole("button", { name: "Connect ChatGPT" }));
    expect(switchSource).not.toHaveBeenCalled();
    const connection = { status: "ready", eligible: true, availableModels: ["gpt-5.4"] };
    setConnection(connection);
    await act(async () =>
      client.setQueryData(factoryCodexConnectionKey("org-1", REFUND_FACTORY.id ?? "factory-1", "me"), connection),
    );
    const confirm = await screen.findByRole("button", { name: "Switch to Codex" });
    await waitFor(() => expect(confirm).toBeEnabled());
    expect(switchSource).not.toHaveBeenCalled();
    await user.click(confirm);
    expect(switchSource).toHaveBeenCalledWith({ source: "subscription", defaultOwnerId: "me", apiKey: undefined });
  });

  it("lets a disconnected admin select another connected member and retry a failed switch", async () => {
    setConnection({ status: "disconnected", eligible: true });
    switchSource.mockRejectedValueOnce(new Error("The intake configuration changed."));
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    await user.click(await screen.findByRole("tab", { name: "Your ChatGPT subscription" }));
    await user.click(screen.getByRole("combobox", { name: "Default task owner (required)" }));
    await user.click(await screen.findByRole("option", { name: "Other member" }));
    await user.click(screen.getByRole("button", { name: "Switch to Codex" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("The intake configuration changed.");
    expect(switchSource).toHaveBeenCalledWith({ source: "subscription", defaultOwnerId: "other", apiKey: undefined });
    await user.click(screen.getByRole("button", { name: "Switch to Codex" }));
    await waitFor(() => expect(switchSource).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("does not switch after owner selection is canceled", async () => {
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    await user.click(await screen.findByRole("tab", { name: "Your ChatGPT subscription" }));
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Cancel" }));
    expect(switchSource).not.toHaveBeenCalled();
    expect(screen.getByRole("tab", { name: "SuperPlane" })).toHaveAttribute("aria-selected", "true");
  });

  it("warns about replacing intake owners and changes the default owner while Codex stays active", async () => {
    setConnection({ status: "ready", eligible: true, availableModels: ["gpt-5.4"] });
    renderPage("AGENT_HARNESS_CODEX", true);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Change default task owner" }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText(/replaces the default task owner of every intake/)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("combobox", { name: "Default task owner (required)" }));
    await user.click(await screen.findByRole("option", { name: "Other member" }));
    await user.click(within(dialog).getByRole("button", { name: "Save default owner" }));
    expect(switchSource).toHaveBeenCalledWith({ source: "subscription", defaultOwnerId: "other", apiKey: undefined });
  });

  it("warns about replacing intake owners before the first switch to Codex", async () => {
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    await user.click(await screen.findByRole("tab", { name: "Your ChatGPT subscription" }));
    expect(await screen.findByText(/replaces the default task owner of every intake/)).toBeInTheDocument();
  });

  it("switches a subscription workspace back to hosted models", async () => {
    setConnection({ status: "ready", eligible: true, availableModels: ["gpt-5.4"] });
    renderPage("AGENT_HARNESS_CODEX", true);
    const user = userEvent.setup();
    await user.click(screen.getByRole("tab", { name: "SuperPlane" }));
    await user.click(screen.getByRole("button", { name: "Switch to SuperPlane" }));
    expect(switchSource).toHaveBeenCalledWith({ source: "hosted", apiKey: undefined });
  });
}
