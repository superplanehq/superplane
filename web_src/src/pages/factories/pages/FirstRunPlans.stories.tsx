import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { FirstRunCardScreen, type FirstRunCardPhase } from "./onboarding/first-run/FirstRunCardScreen";
import { FirstRunCardReturnScreen } from "./onboarding/first-run/FirstRunCardReturnScreen";
import { firstRunStoryChrome } from "./onboarding/first-run/firstRunMocks";
import { repositorySphereFor } from "./onboarding/first-run/firstRunSphereFor";
import { FirstRunTicketsScreen } from "./onboarding/first-run/FirstRunTicketsScreen";

const plans = [
  { key: "pay_as_you_go", name: "Pay-as-you-go", priceCents: "0", parallelTaskLimit: 5 },
  { key: "starter", name: "Starter", priceCents: "2000", subscriptionRequired: true },
];

function PlanPreview({
  initialPhase = "add-card",
  error,
  welcomeCreditCents = 1000,
}: {
  initialPhase?: FirstRunCardPhase;
  error?: string;
  welcomeCreditCents?: number;
}) {
  const [selectedPlanKey, setSelectedPlanKey] = useState("pay_as_you_go");
  const [phase, setPhase] = useState(initialPhase);
  const [tickets, setTickets] = useState(false);
  const sphere = repositorySphereFor("acme/payments-service");
  if (tickets) {
    return (
      <FirstRunTicketsScreen
        ticketSource="github-issues"
        repository="acme/payments-service"
        githubConnected
        chrome={firstRunStoryChrome(3)}
        sphere={sphere}
        continueLabel="Continue"
        onSelectTicketSource={() => console.log("select ticket source")}
        onAnalyzeTickets={() => setTickets(false)}
      />
    );
  }
  return (
    <FirstRunCardScreen
      plans={plans}
      welcomeCreditCents={welcomeCreditCents}
      selectedPlanKey={selectedPlanKey}
      phase={phase}
      error={error}
      chrome={{ ...firstRunStoryChrome(4), onBack: () => setTickets(true) }}
      sphere={sphere}
      onSelectPlan={(key) => {
        console.log("select plan", key);
        setSelectedPlanKey(key);
      }}
      onContinue={() => {
        console.log("continue with", selectedPlanKey);
        setPhase(phase === "card-saved" ? "plan-active" : "card-saved");
      }}
    />
  );
}

const meta = {
  title: "Factories/Pages/First run plans",
  parameters: { layout: "fullscreen" },
} satisfies Meta;
export default meta;
type Story = StoryObj<typeof meta>;

export const ChoosePlan: Story = { render: () => <PlanPreview /> };
export const CardSaved: Story = { render: () => <PlanPreview initialPhase="card-saved" /> };
export const FinishingSetup: Story = {
  render: () => (
    <FirstRunCardReturnScreen
      chrome={firstRunStoryChrome(4)}
      sphere={repositorySphereFor("acme/payments-service")}
      busy
      onRetry={() => console.log("retry setup")}
      onBack={() => console.log("back to plans")}
    />
  ),
};
export const FinishSetupError: Story = {
  render: () => (
    <FirstRunCardReturnScreen
      chrome={firstRunStoryChrome(4)}
      sphere={repositorySphereFor("acme/payments-service")}
      busy={false}
      error="Your subscription is not active yet. Wait a moment, then try again."
      onRetry={() => console.log("retry setup")}
      onBack={() => console.log("back to plans")}
    />
  ),
};
export const CardError: Story = {
  render: () => <PlanPreview error="Could not open card setup. Try again." />,
};

export const NoSignupCredit: Story = { render: () => <PlanPreview welcomeCreditCents={0} /> };
