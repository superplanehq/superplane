import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

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
      remainingCreditCents: 0,
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
  it("shows hosted credit when the organization has a grant", () => {
    renderAgentStep();

    expect(screen.getByTestId("hosted-credit-grant")).toBeInTheDocument();
    expect(screen.getByText("SuperPlane-hosted credit")).toBeInTheDocument();
    expect(
      screen.getByText(
        "This organization has $41.24 of hosted credit. You can continue without connecting your own keys.",
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

  it("shows Anthropic, OpenAI, and OpenRouter as connectable, and Cursor as coming soon", () => {
    renderAgentStep({ spend: { remainingCreditCents: "0", grantTotalCents: "0" } });

    expect(screen.getByRole("button", { name: "Connect Anthropic" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Connect OpenAI" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Connect OpenRouter" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Connect Cursor/ })).not.toBeInTheDocument();
    expect(screen.getByText("Coming soon")).toBeInTheDocument();
  });

  it("asks to connect a provider when a grant exists but remaining credit is empty", () => {
    renderAgentStep({
      spend: { remainingCreditCents: "0", grantTotalCents: "5000" },
    });

    expect(screen.getByTestId("hosted-credit-grant")).toBeInTheDocument();
    expect(screen.getByText("Hosted credit is empty. Connect a provider to continue.")).toBeInTheDocument();
  });

  it("connects OpenAI without selecting a single harness", async () => {
    const user = userEvent.setup();
    const { onRequestConnect } = renderAgentStep({
      spend: { remainingCreditCents: "0", grantTotalCents: "0" },
    });

    await user.click(screen.getByRole("button", { name: "Connect OpenAI" }));

    expect(onRequestConnect).toHaveBeenCalledWith("openai");
  });
});
