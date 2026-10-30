import { QueryClient } from "@tanstack/react-query";
import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "bun:test";
import {
  factoriesListFactoryOwnerEligibility,
  meCancelCodexConnection,
  type FactoriesFactoryCodexConnection,
} from "@/api-client";
import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { factoryCodexConnectionKey } from "../onboarding/useFactoryCodexConnection";
import { renderPage } from "./organizationLLMModels.spec.render";

function expectSingleDialog(name: string) {
  expect(screen.getAllByRole("dialog", { hidden: true })).toHaveLength(1);
  expect(screen.getByRole("dialog", { name })).toBeInTheDocument();
}

export function subscriptionModelSourceSwitchCases({
  setConnection,
  switchSource,
}: {
  setConnection: (connection: FactoriesFactoryCodexConnection) => void;
  switchSource: ReturnType<typeof vi.fn>;
}) {
  it("returns from sign-in to owner selection and requires explicit confirmation", async () => {
    setConnection({ status: "disconnected", eligible: true });
    let connected = false;
    vi.mocked(factoriesListFactoryOwnerEligibility).mockImplementation(
      async () =>
        ({
          data: {
            subscription: true,
            ownershipChangeAllowed: true,
            members: [{ userId: "me", eligible: connected }],
          },
        }) as never,
    );
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    renderPage("AGENT_HARNESS_SUPERPLANE", false, client);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Change model source" }));
    await user.click(screen.getByRole("button", { name: "Select ChatGPT subscriptions" }));
    expectSingleDialog("Switch to Codex");
    expect(screen.getByRole("button", { name: "Switch to Codex" })).toBeDisabled();
    await waitFor(() => expect(screen.queryAllByRole("combobox").length).toBe(0));
    await user.click(await screen.findByRole("button", { name: "Connect Codex" }));
    expectSingleDialog("Connect Codex");
    await user.click(await screen.findByRole("button", { name: "Connect ChatGPT" }));
    const connection = { status: "ready", eligible: true, availableModels: ["gpt-5.4"] };
    connected = true;
    setConnection(connection);
    await act(async () =>
      client.setQueryData(factoryCodexConnectionKey("org-1", REFUND_FACTORY.id ?? "factory-1", "me"), connection),
    );
    await waitFor(() => expect(screen.getByRole("button", { name: "Switch to Codex" })).toBeEnabled());
    expectSingleDialog("Switch to Codex");
    expect(screen.getByRole("combobox", { name: "Default task owner (required)" })).toBeEnabled();
    expect(switchSource).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Switch to Codex" }));
    expect(switchSource).toHaveBeenCalledWith({ source: "subscription", defaultOwnerId: "me", apiKey: undefined });
    expect(await screen.findByRole("status", { name: "ChatGPT subscriptions is active" })).toBeInTheDocument();
  });

  it("lets a disconnected admin select another connected member and retry a failed switch", async () => {
    setConnection({ status: "disconnected", eligible: true });
    switchSource.mockRejectedValueOnce(new Error("The intake configuration changed."));
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Change model source" }));
    await user.click(screen.getByRole("button", { name: "Select ChatGPT subscriptions" }));
    await user.click(screen.getByRole("combobox", { name: "Default task owner (required)" }));
    await user.click(await screen.findByRole("option", { name: "Other member" }));
    await user.click(screen.getByRole("button", { name: "Switch to Codex" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("The intake configuration changed.");
    expect(switchSource).toHaveBeenCalledWith({ source: "subscription", defaultOwnerId: "other", apiKey: undefined });
    await user.click(screen.getByRole("button", { name: "Switch to Codex" }));
    await waitFor(() => expect(switchSource).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("cancels pending sign-in and returns to owner selection without switching", async () => {
    setConnection({ status: "pending", eligible: true, reservationHeld: true });
    vi.mocked(meCancelCodexConnection).mockImplementationOnce(async () => {
      const connection = { status: "disconnected", eligible: true };
      setConnection(connection);
      return { data: { connection } } as never;
    });
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Change model source" }));
    await user.click(screen.getByRole("button", { name: "Select ChatGPT subscriptions" }));
    await user.click(await screen.findByRole("button", { name: "Connect Codex" }));
    await screen.findByText("Wait for the ChatGPT sign-in code.");
    expectSingleDialog("Connect Codex");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(meCancelCodexConnection).toHaveBeenCalledTimes(1);
    await screen.findByRole("dialog", { name: "Switch to Codex" });
    expectSingleDialog("Switch to Codex");
    expect(switchSource).not.toHaveBeenCalled();
    expect(screen.getByRole("status", { name: "SuperPlane is active", hidden: true })).toBeInTheDocument();
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Cancel" }));
    expect(switchSource).not.toHaveBeenCalled();
    expect(screen.getByRole("status", { name: "SuperPlane is active" })).toBeInTheDocument();
  });

  it("warns about replacing intake owners and changes the default owner while Codex stays active", async () => {
    setConnection({ status: "ready", eligible: true, availableModels: ["gpt-5.4"] });
    renderPage("AGENT_HARNESS_CODEX", true);
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Options for Other member" }));
    await user.click(screen.getByRole("menuitem", { name: "Set as default task owner" }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText(/Replaces every default owner, including scheduled intakes/)).toBeInTheDocument();
    expect(within(dialog).getByRole("combobox", { name: "Default task owner (required)" })).toHaveTextContent(
      "Other member",
    );
    await user.click(within(dialog).getByRole("button", { name: "Save default owner" }));
    expect(switchSource).toHaveBeenCalledWith({ source: "subscription", defaultOwnerId: "other", apiKey: undefined });
  });

  it("warns about replacing intake owners before the first switch to Codex", async () => {
    renderPage("AGENT_HARNESS_SUPERPLANE");
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: "Change model source" }));
    await user.click(screen.getByRole("button", { name: "Select ChatGPT subscriptions" }));
    expect(await screen.findByText(/Replaces every default owner, including scheduled intakes/)).toBeInTheDocument();
  });

  it("switches a subscription workspace back to hosted models", async () => {
    setConnection({ status: "ready", eligible: true, availableModels: ["gpt-5.4"] });
    renderPage("AGENT_HARNESS_CODEX", true);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Change model source" }));
    await user.click(screen.getByRole("button", { name: "Select SuperPlane" }));
    await user.click(screen.getByRole("button", { name: "Switch to SuperPlane" }));
    expect(switchSource).toHaveBeenCalledWith({ source: "hosted", apiKey: undefined });
    expect(await screen.findByRole("status", { name: "SuperPlane is active" })).toBeInTheDocument();
  });
}
