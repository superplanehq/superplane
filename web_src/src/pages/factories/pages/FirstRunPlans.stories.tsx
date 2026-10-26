import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { FirstRunCardScreen, type FirstRunCardPhase } from "./onboarding/first-run/FirstRunCardScreen";
import { firstRunStoryChrome } from "./onboarding/first-run/firstRunMocks";
import { repositorySphereFor } from "./onboarding/first-run/firstRunSphereFor";
import { FirstRunTicketsScreen } from "./onboarding/first-run/FirstRunTicketsScreen";

const plans = [
  { key: "pay_as_you_go", name: "Pay-as-you-go", priceCents: "0" },
  { key: "starter", name: "Starter", priceCents: "2000", subscriptionRequired: true },
];

function PlanPreview({ initialPhase = "add-card", error }: { initialPhase?: FirstRunCardPhase; error?: string }) {
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
export const WaitingForCard: Story = { render: () => <PlanPreview initialPhase="waiting-for-card" /> };
export const CardError: Story = {
  render: () => <PlanPreview error="Allow pop-ups for SuperPlane, then try again." />,
};
