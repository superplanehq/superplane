import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { FACTORIES_ORGANIZATION_ID } from "@/pages/factories/__fixtures__/factoryPageResponses";

import { AgentStep } from "./AgentStep";
import type { IntegrationId } from "./onboardingFixtures";
import { useOnboardingSetupState } from "./useOnboardingSetupState";

const spendState: { remainingCreditCents: string; grantTotalCents: string } = {
  remainingCreditCents: "4124",
  grantTotalCents: "5000",
};

vi.mock("@/hooks/useOrganizationLLMSpend", () => ({
  useOrganizationLLMSpend: () => ({ data: spendState }),
}));

function renderAgentStep(args?: {
  connected?: IntegrationId[];
  spend?: { remainingCreditCents: string; grantTotalCents: string };
  onRequestConnect?: (id: IntegrationId) => void;
}) {
  Object.assign(spendState, args?.spend ?? { remainingCreditCents: "4124", grantTotalCents: "5000" });
  const onRequestConnect = args?.onRequestConnect ?? vi.fn();

  function Harness() {
    const setup = useOnboardingSetupState("Payments", {
      connected: new Set(args?.connected ?? []),
      remainingCreditCents: Number(spendState.remainingCreditCents),
      simulateDiscovery: false,
    });
    return <AgentStep organizationId={FACTORIES_ORGANIZATION_ID} setup={setup} onRequestConnect={onRequestConnect} />;
  }

  return {
    onRequestConnect,
    ...render(<Harness />),
  };
}

describe("AgentStep", () => {
  beforeEach(() => {
    sessionStorage.removeItem("superplane.onboarding.keyProvider");
  });

  it("selects SuperPlane agent by default and hides BYOK rows", () => {
    renderAgentStep();

    expect(screen.getByRole("button", { name: /SuperPlane agent/ })).toBeInTheDocument();
    expect(
      screen.getByText("SuperPlane will run the agent on this workspace. Work starts only after you approve a ticket."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use your own key" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect Anthropic" })).not.toBeInTheDocument();
    expect(screen.queryByText("OpenRouter")).not.toBeInTheDocument();
    expect(screen.queryByTestId("hosted-credit-grant")).not.toBeInTheDocument();
  });

  it("expands Claude, OpenAI, and OpenRouter after Use your own key, without Cursor", async () => {
    const user = userEvent.setup();
    renderAgentStep({ spend: { remainingCreditCents: "0", grantTotalCents: "0" } });

    await user.click(screen.getByRole("button", { name: "Use your own key" }));

    expect(screen.getByRole("button", { name: "Connect Anthropic" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Connect OpenAI" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Connect OpenRouter" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Connect Cursor/ })).not.toBeInTheDocument();
    expect(screen.queryByText("Coming soon")).not.toBeInTheDocument();
  });

  it("shows empty SuperPlane agent credit copy on the BYOK path", async () => {
    const user = userEvent.setup();
    renderAgentStep({
      spend: { remainingCreditCents: "0", grantTotalCents: "5000" },
    });

    expect(
      screen.queryByText("SuperPlane agent credit is empty. Connect a provider to continue."),
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Use your own key" }));

    expect(screen.getByText("SuperPlane agent credit is empty. Connect a provider to continue.")).toBeInTheDocument();
  });

  it("connects OpenAI from the expanded BYOK list", async () => {
    const user = userEvent.setup();
    const { onRequestConnect } = renderAgentStep({
      spend: { remainingCreditCents: "0", grantTotalCents: "0" },
    });

    await user.click(screen.getByRole("button", { name: "Use your own key" }));
    await user.click(screen.getByRole("button", { name: "Connect OpenAI" }));

    expect(onRequestConnect).toHaveBeenCalledWith("openai");
  });
});
