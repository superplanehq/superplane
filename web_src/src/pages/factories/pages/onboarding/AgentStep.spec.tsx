import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FACTORIES_ORGANIZATION_ID } from "@/pages/factories/__fixtures__/factoryPageResponses";

import { AgentStep, type OnboardingAgentModelPicker } from "./AgentStep";
import type { IntegrationId } from "./onboardingFixtures";
import { useOnboardingSetupState } from "./useOnboardingSetupState";

const spendState: { remainingCreditCents: string; grantTotalCents: string; billingEnabled: boolean } = {
  remainingCreditCents: "4124",
  grantTotalCents: "5000",
  billingEnabled: true,
};

vi.mock("@/hooks/useOrganizationWorkspaceUsage", () => ({
  useOrganizationWorkspaceUsage: () => ({ data: spendState }),
}));

function renderAgentStep(args?: {
  connected?: IntegrationId[];
  spend?: { remainingCreditCents: string; grantTotalCents: string; billingEnabled?: boolean };
  showCustomProvider?: boolean;
  onRequestConnect?: (id: IntegrationId) => void;
  modelPicker?: OnboardingAgentModelPicker;
}) {
  const spend = args?.spend ?? { remainingCreditCents: "4124", grantTotalCents: "5000" };
  spendState.remainingCreditCents = spend.remainingCreditCents;
  spendState.grantTotalCents = spend.grantTotalCents;
  spendState.billingEnabled = spend.billingEnabled ?? true;
  const onRequestConnect = args?.onRequestConnect ?? vi.fn();

  function Harness() {
    const setup = useOnboardingSetupState("Payments", {
      connected: new Set(args?.connected ?? []),
      remainingCreditCents: 0,
      simulateDiscovery: false,
    });
    return (
      <AgentStep
        organizationId={FACTORIES_ORGANIZATION_ID}
        setup={setup}
        showCustomProvider={args?.showCustomProvider}
        onRequestConnect={onRequestConnect}
        modelPicker={args?.modelPicker}
      />
    );
  }

  return {
    onRequestConnect,
    ...render(<Harness />),
  };
}

describe("AgentStep", () => {
  it("lets the user select a model from the connected key", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderAgentStep({
      connected: ["claude"],
      modelPicker: {
        models: ["claude-sonnet-4-6", "claude-opus-5-5"],
        value: "claude-sonnet-4-6",
        loading: false,
        error: false,
        retry: vi.fn(),
        onChange,
      },
    });

    await user.click(screen.getByRole("combobox", { name: "Model" }));
    await user.click(screen.getByRole("option", { name: "claude-opus-5-5" }));

    expect(onChange).toHaveBeenCalledWith("claude-opus-5-5");
  });

  it("offers a retry when key models cannot load", async () => {
    const user = userEvent.setup();
    const retry = vi.fn();
    renderAgentStep({
      connected: ["claude"],
      modelPicker: {
        models: [],
        value: "",
        loading: false,
        error: true,
        retry,
        onChange: vi.fn(),
      },
    });

    expect(screen.getByRole("combobox", { name: "Model" })).toBeDisabled();
    expect(screen.getByText("Unable to load models. Try again.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    expect(retry).toHaveBeenCalledTimes(1);
  });
  it("shows hosted credit when the organization has a grant", () => {
    renderAgentStep();

    expect(screen.getByTestId("hosted-credit-grant")).toBeInTheDocument();
    expect(screen.getByText("SuperPlane-hosted credit")).toBeInTheDocument();
    expect(
      screen.getByText(
        "This organization has $41.24 of trial usage for machines and managed models. Contact an installation admin to keep hosted runs after the trial.",
      ),
    ).toBeInTheDocument();
  });

  it("hides the grant block when the organization has no grant", () => {
    renderAgentStep({
      spend: { remainingCreditCents: "0", grantTotalCents: "0" },
    });

    expect(screen.getByRole("button", { name: "Connect Anthropic" })).toBeInTheDocument();
    expect(screen.queryByTestId("hosted-credit-grant")).not.toBeInTheDocument();
  });

  it("hides the grant block when product billing is off", () => {
    renderAgentStep({
      spend: { remainingCreditCents: "5000", grantTotalCents: "5000", billingEnabled: false },
    });

    expect(screen.queryByTestId("hosted-credit-grant")).not.toBeInTheDocument();
    expect(screen.queryByText("SuperPlane-hosted credit")).not.toBeInTheDocument();
  });

  it("shows Anthropic, OpenAI, and OpenRouter as connectable", () => {
    renderAgentStep({ spend: { remainingCreditCents: "0", grantTotalCents: "0" } });

    expect(screen.getByRole("button", { name: "Connect Anthropic" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Connect OpenAI" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Connect OpenRouter" })).toBeInTheDocument();
    expect(screen.queryByText("Cursor")).not.toBeInTheDocument();
    expect(screen.queryByText("Coming soon")).not.toBeInTheDocument();
  });

  it("asks to connect a provider when a grant exists but remaining credit is empty", () => {
    renderAgentStep({
      spend: { remainingCreditCents: "0", grantTotalCents: "5000" },
    });

    expect(screen.getByTestId("hosted-credit-grant")).toBeInTheDocument();
    expect(
      screen.getByText("Trial credit is used up. Contact an installation admin or connect a provider to continue."),
    ).toBeInTheDocument();
  });

  it("connects OpenAI without selecting a single harness", async () => {
    const user = userEvent.setup();
    const { onRequestConnect } = renderAgentStep({
      spend: { remainingCreditCents: "0", grantTotalCents: "0" },
    });

    await user.click(screen.getByRole("button", { name: "Connect OpenAI" }));

    expect(onRequestConnect).toHaveBeenCalledWith("openai");
  });

  it("hides the custom provider unless the organization can use one", () => {
    renderAgentStep({ spend: { remainingCreditCents: "0", grantTotalCents: "0" } });

    expect(screen.queryByRole("button", { name: "Connect Custom provider" })).not.toBeInTheDocument();
  });

  it("connects a custom provider when the organization can use one", async () => {
    const user = userEvent.setup();
    const { onRequestConnect } = renderAgentStep({
      spend: { remainingCreditCents: "0", grantTotalCents: "0" },
      showCustomProvider: true,
    });

    expect(screen.getByText("Set a provider URL, token, and API type.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Connect Custom provider" }));

    expect(onRequestConnect).toHaveBeenCalledWith("customLlm");
  });
});
