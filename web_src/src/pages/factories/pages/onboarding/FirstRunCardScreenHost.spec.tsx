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
const plan = { key: "pay_as_you_go", name: "Pay-as-you-go", priceCents: "0" };
const starter = { key: "starter", name: "Starter", priceCents: "2000", subscriptionRequired: true };

function cardStep(overrides: Partial<OnboardingCardStep> = {}): OnboardingCardStep {
  return {
    ready: true,
    available: true,
    required: true,
    plans: [plan, starter],
    planActive: false,
    hasPaymentMethod: false,
    refetch: vi.fn(async () => undefined),
    ...overrides,
  };
}

function renderHost(step: OnboardingCardStep) {
  const finishSetup = vi.fn(async () => {});
  render(
    <FirstRunCardScreenHost
      organizationId="org-1"
      flow={{ busy: false, cardStep: step, finishSetup }}
      chrome={chrome}
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

it("saves a card, starts Pay-as-you-go, and continues setup", async () => {
  const step = cardStep();
  const { finishSetup } = renderHost(step);
  expect(screen.getByRole("heading", { name: "Choose your starting plan" })).toBeInTheDocument();
  expect(screen.getByRole("radio", { name: /Pay-as-you-go/ })).toBeChecked();
  expect(screen.getByRole("radio", { name: /Starter/ })).not.toBeChecked();
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
      expect.objectContaining({ returnPath: expect.stringContaining("plan=pay_as_you_go") }),
    ),
  );
  await waitFor(() => expect(finishSetup).toHaveBeenCalledTimes(1));
  expect(step.refetch).toHaveBeenCalled();
});

it("starts the plan at once when a card is already saved", async () => {
  const { finishSetup } = renderHost(cardStep({ hasPaymentMethod: true }));
  expect(screen.getByText(/Your card is saved/)).toBeInTheDocument();
  await userEvent.setup().click(screen.getByTestId("first-run-card-continue"));
  expect(cardSetup.open).not.toHaveBeenCalled();
  await waitFor(() => expect(finishSetup).toHaveBeenCalledTimes(1));
  expect(activateOnboardingPlan).toHaveBeenCalledWith(
    "org-1",
    "pay_as_you_go",
    expect.objectContaining({ returnPath: expect.stringContaining("plan=pay_as_you_go") }),
  );
});

it("uses the selected Starter plan after card setup and waits for checkout", async () => {
  activateOnboardingPlan.mockResolvedValue(true);
  const { finishSetup } = renderHost(cardStep());
  const user = userEvent.setup();
  await user.click(screen.getByRole("radio", { name: /Starter/ }));
  await user.click(screen.getByRole("button", { name: "Continue with Starter" }));
  expect(activateOnboardingPlan).not.toHaveBeenCalled();
  cardSetup.open.mock.calls[0][0]?.();
  await waitFor(() =>
    expect(activateOnboardingPlan).toHaveBeenCalledWith(
      "org-1",
      "starter",
      expect.objectContaining({ returnPath: expect.stringContaining("plan=starter") }),
    ),
  );
  expect(finishSetup).not.toHaveBeenCalled();
});

it("keeps the selected plan locked while card setup is open", () => {
  cardSetup.waitingForCard = true;
  renderHost(cardStep());
  for (const option of screen.getAllByRole("radio")) expect(option).toBeDisabled();
});

it("does not allow activation when no starter plans are available", () => {
  renderHost(cardStep({ plans: [] }));
  expect(screen.getByRole("button", { name: "Continue" })).toBeDisabled();
  expect(screen.getByRole("alert")).toHaveTextContent("No starter plans are available");
});

it("continues without a plan change when the plan is already active", async () => {
  const { finishSetup } = renderHost(cardStep({ required: false, planActive: true, hasPaymentMethod: true }));
  expect(screen.getByText("Your plan is active. Continue to finish setup.")).toBeInTheDocument();
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
