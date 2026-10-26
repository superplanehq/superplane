import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "bun:test";

import { FirstRunCardScreenHost } from "./FirstRunCardScreenHost";
import type { activateOnboardingPlan as activateOnboardingPlanFn } from "./onboardingCardPlan";
import type { OnboardingCardStep } from "./useOnboardingCardStep";

const cardSetup = {
  open: vi.fn<(onSaved?: () => void) => void>(),
  error: "",
  isPending: false,
  opening: false,
  waitingForCard: false,
};
vi.mock("@/hooks/useBillingCardSetup", () => ({ useBillingCardSetup: () => ({ ...cardSetup }) }));
const activateOnboardingPlan = vi.fn<typeof activateOnboardingPlanFn>(async () => false);
vi.mock("./onboardingCardPlan", () => ({ activateOnboardingPlan }));

const chrome = { stepIndex: 4, stepCount: 6 };
const plan = { key: "pay_as_you_go", name: "Pay as you go", onboardingDefault: true, trialCreditCents: "1000" };

function cardStep(overrides: Partial<OnboardingCardStep> = {}): OnboardingCardStep {
  return {
    ready: true,
    available: true,
    required: true,
    plan,
    planActive: false,
    hasPaymentMethod: false,
    refetch: vi.fn(async () => undefined),
    ...overrides,
  };
}

const summary = {
  repository: "acme/payments",
  ticketSource: "github-issues" as const,
  initialImportCount: 3,
  automaticIntake: true,
};

function renderHost(step: OnboardingCardStep, summaryOverrides: Partial<typeof summary> = {}) {
  const finishSetup = vi.fn(async () => {});
  render(
    <FirstRunCardScreenHost
      organizationId="org-1"
      flow={{ busy: false, cardStep: step, finishSetup }}
      chrome={chrome}
      summary={{ ...summary, ...summaryOverrides }}
    />,
  );
  return { finishSetup };
}

beforeEach(() => {
  cardSetup.open.mockReset();
  cardSetup.error = "";
  cardSetup.opening = false;
  cardSetup.waitingForCard = false;
  activateOnboardingPlan.mockReset().mockResolvedValue(false);
});

it("saves a card, starts the default plan with its credit, and continues setup", async () => {
  const step = cardStep();
  const { finishSetup } = renderHost(step);
  expect(screen.getByRole("heading", { name: "Start with $10 in credit" })).toBeInTheDocument();
  expect(screen.getByText(/\$10 on the Pay as you go plan/)).toBeInTheDocument();
  expect(screen.getByText("Polar saves your card. SuperPlane does not charge it.")).toBeInTheDocument();
  expect(screen.getByTestId("first-run-step-repository")).toHaveTextContent("Repository: acme/payments");
  expect(screen.getByTestId("first-run-step-tickets")).toHaveTextContent("Imports your 3 newest GitHub issues");
  expect(screen.getByTestId("first-run-step-card")).toHaveAttribute("data-state", "active");
  expect(screen.queryByRole("button", { name: /skip/i })).not.toBeInTheDocument();
  expect(screen.getByLabelText("Step 5 of 6")).toBeInTheDocument();

  await userEvent.setup().click(screen.getByTestId("first-run-add-card"));
  expect(cardSetup.open).toHaveBeenCalledTimes(1);
  expect(activateOnboardingPlan).not.toHaveBeenCalled();

  cardSetup.open.mock.calls[0][0]?.();
  await waitFor(() =>
    expect(activateOnboardingPlan).toHaveBeenCalledWith(
      "org-1",
      "pay_as_you_go",
      expect.objectContaining({ returnPath: `${window.location.pathname}${window.location.search}` }),
    ),
  );
  await waitFor(() => expect(finishSetup).toHaveBeenCalledTimes(1));
  expect(step.refetch).toHaveBeenCalled();
});

it("starts the plan at once when a card is already saved", async () => {
  const { finishSetup } = renderHost(cardStep({ hasPaymentMethod: true }));
  expect(screen.getByText("Your card is saved.")).toBeInTheDocument();
  await userEvent.setup().click(screen.getByTestId("first-run-card-continue"));
  expect(cardSetup.open).not.toHaveBeenCalled();
  await waitFor(() => expect(finishSetup).toHaveBeenCalledTimes(1));
  expect(activateOnboardingPlan).toHaveBeenCalledWith(
    "org-1",
    "pay_as_you_go",
    expect.objectContaining({ returnPath: `${window.location.pathname}${window.location.search}` }),
  );
});

it("describes the intake when the first import is off", () => {
  renderHost(cardStep(), { initialImportCount: 0, automaticIntake: true });
  expect(screen.getByTestId("first-run-step-tickets")).toHaveTextContent("Imports new GitHub issues only");
});

it("checks out a per-seat plan for the occupied seats and does not finish setup", async () => {
  activateOnboardingPlan.mockResolvedValue(true);
  const { finishSetup } = renderHost(
    cardStep({
      hasPaymentMethod: true,
      occupiedSeats: 2,
      plan: { key: "team", name: "Team", onboardingDefault: true, perSeat: true, subscriptionRequired: true },
    }),
  );
  expect(screen.getByRole("heading", { name: "Start on the Team plan" })).toBeInTheDocument();
  expect(screen.getByText("Add a payment card, then complete checkout for the Team plan.")).toBeInTheDocument();
  await userEvent.setup().click(screen.getByTestId("first-run-card-continue"));
  await waitFor(() =>
    expect(activateOnboardingPlan).toHaveBeenCalledWith("org-1", "team", expect.objectContaining({ seats: 2 })),
  );
  expect(finishSetup).not.toHaveBeenCalled();
});

it("continues without a plan change when the plan is already active", async () => {
  const { finishSetup } = renderHost(cardStep({ required: false, planActive: true, hasPaymentMethod: true }));
  expect(screen.getByText("Your organization is on the Pay as you go plan.")).toBeInTheDocument();
  await userEvent.setup().click(screen.getByTestId("first-run-card-continue"));
  await waitFor(() => expect(finishSetup).toHaveBeenCalledTimes(1));
  expect(activateOnboardingPlan).not.toHaveBeenCalled();
});

it("keeps the step open with the error when the plan cannot start", async () => {
  activateOnboardingPlan.mockRejectedValue(new Error("Could not select this plan."));
  const { finishSetup } = renderHost(cardStep({ hasPaymentMethod: true }));
  await userEvent.setup().click(screen.getByTestId("first-run-card-continue"));
  expect(await screen.findByRole("alert")).toHaveTextContent("Could not select this plan.");
  expect(finishSetup).not.toHaveBeenCalled();
  expect(screen.getByTestId("first-run-card-continue")).toBeEnabled();
});

it("shows the card setup error and the waiting state", () => {
  cardSetup.error = "Allow pop-ups for SuperPlane, then try again.";
  cardSetup.waitingForCard = true;
  renderHost(cardStep());
  expect(screen.getByRole("alert")).toHaveTextContent("Allow pop-ups for SuperPlane, then try again.");
  expect(screen.getByRole("button", { name: "Waiting for the card…" })).toBeDisabled();
});
