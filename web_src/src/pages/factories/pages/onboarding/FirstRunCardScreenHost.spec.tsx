import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, expect, it, vi } from "bun:test";
import { FirstRunCardScreenHost } from "./FirstRunCardScreenHost";
import type { OnboardingCardStep } from "./useOnboardingCardStep";

const openCardSetup = vi.fn(async (_org: string) => ({ token: "polar_cst_test", origin: "https://sandbox.polar.sh" }));
const activateOnboardingPlan = vi.fn(async (..._args: unknown[]) => false);
const resumeOnboardingPlan = vi.fn(async (..._args: unknown[]) => false);
vi.mock("./onboardingCardPlan", () => ({
  activateOnboardingPlan,
  createOnboardingCardSession: openCardSetup,
  resumeOnboardingPlan,
}));
const chrome = { stepIndex: 4, stepCount: 6 };
const plans = [
  { key: "pay_as_you_go", name: "Pay-as-you-go", priceCents: "0", parallelTaskLimit: 5 },
  { key: "starter", name: "Starter", priceCents: "2000", subscriptionRequired: true },
];
function cardStep(overrides: Partial<OnboardingCardStep> = {}): OnboardingCardStep {
  return {
    ready: true,
    available: true,
    required: true,
    requiresAdmin: false,
    plans,
    planActive: false,
    hasPaymentMethod: false,
    refetch: vi.fn(async () => undefined),
    ...overrides,
  };
}
function renderHost(step: OnboardingCardStep) {
  const finishSetup = vi.fn(async () => {});
  const props = { organizationId: "org-1", flow: { busy: false, cardStep: step, finishSetup }, chrome };
  return { ...render(<FirstRunCardScreenHost {...props} />), props, finishSetup };
}
beforeEach(() => {
  window.history.replaceState({}, "", "/org-1/setup?step=card");
  openCardSetup.mockReset().mockResolvedValue({ token: "polar_cst_test", origin: "https://sandbox.polar.sh" });
  activateOnboardingPlan.mockReset().mockResolvedValue(false);
  resumeOnboardingPlan.mockReset().mockResolvedValue(false);
});

it.each([1, 2])("shows the saved task limit of %i instead of inferring it from the plan name", (limit) => {
  renderHost(
    cardStep({
      plans: [
        { ...plans[0], parallelTaskLimit: undefined },
        { ...plans[1], parallelTaskLimit: limit },
      ],
    }),
  );
  expect(screen.getByText(`Up to ${limit} ${limit === 1 ? "task" : "tasks"} at once`)).toBeInTheDocument();
  expect(screen.getByText("Unlimited tasks at once")).toBeInTheDocument();
  expect(screen.queryByText("Up to 5 tasks at once")).not.toBeInTheDocument();
});

it("sends Starter directly to checkout without a separate card setup", async () => {
  activateOnboardingPlan.mockResolvedValue(true);
  const { finishSetup } = renderHost(cardStep());
  expect(screen.getByRole("radio", { name: "Pay-as-you-go" })).toBeChecked();
  await userEvent.setup().click(screen.getByRole("radio", { name: "Starter" }));
  await userEvent.setup().click(screen.getByRole("button", { name: "Continue with Starter" }));
  await waitFor(() =>
    expect(activateOnboardingPlan).toHaveBeenCalledWith("org-1", "starter", {
      returnPath: "/org-1/setup?step=card&plan=starter&card_setup=1",
    }),
  );
  expect(openCardSetup).not.toHaveBeenCalled();
  expect(finishSetup).not.toHaveBeenCalled();
});

it("activates a saved-card plan and finishes without opening Polar", async () => {
  const { finishSetup } = renderHost(cardStep({ hasPaymentMethod: true }));
  await userEvent.setup().click(screen.getByTestId("first-run-card-continue"));
  await waitFor(() => expect(finishSetup).toHaveBeenCalledTimes(1));
  expect(activateOnboardingPlan).toHaveBeenCalledWith("org-1", "pay_as_you_go", expect.any(Object));
  expect(openCardSetup).not.toHaveBeenCalled();
});

it("automatically verifies the Polar return once before finishing", async () => {
  window.history.replaceState({}, "", "/org-1/setup?step=card&plan=pay_as_you_go&card_setup=1");
  let resolve!: (checkout: boolean) => void;
  resumeOnboardingPlan.mockImplementation(
    () =>
      new Promise<boolean>((done) => {
        resolve = done;
      }),
  );
  const { finishSetup, rerender, props } = renderHost(cardStep());
  expect(screen.getByRole("heading", { name: "Finishing setup…" })).toBeInTheDocument();
  expect(screen.queryByRole("radio")).not.toBeInTheDocument();
  expect(finishSetup).not.toHaveBeenCalled();
  rerender(<FirstRunCardScreenHost {...props} />);
  expect(resumeOnboardingPlan).toHaveBeenCalledTimes(1);
  await act(async () => resolve(false));
  await waitFor(() => expect(finishSetup).toHaveBeenCalledTimes(1));
});

it("waits for billing before resuming and preserves Starter across the return", async () => {
  window.history.replaceState({}, "", "/org-1/setup?step=card&plan=starter&card_setup=1&subscribed=1");
  const { rerender, props, finishSetup } = renderHost(cardStep({ ready: false }));
  expect(resumeOnboardingPlan).not.toHaveBeenCalled();
  rerender(<FirstRunCardScreenHost {...props} flow={{ ...props.flow, cardStep: cardStep() }} />);
  await waitFor(() => expect(finishSetup).toHaveBeenCalledTimes(1));
  expect(resumeOnboardingPlan).toHaveBeenCalledWith(
    "org-1",
    "starter",
    "/org-1/setup?step=card&plan=starter&card_setup=1",
    true,
  );
});

it("does not finish while Starter leaves for subscription checkout", async () => {
  window.history.replaceState({}, "", "/org-1/setup?step=card&plan=starter&card_setup=1");
  resumeOnboardingPlan.mockResolvedValue(true);
  const { finishSetup } = renderHost(cardStep());
  await waitFor(() => expect(resumeOnboardingPlan).toHaveBeenCalledTimes(1));
  expect(finishSetup).not.toHaveBeenCalled();
});

it("offers a retry after a failed return without automatic retry loops", async () => {
  window.history.replaceState({}, "", "/org-1/setup?step=card&plan=starter&subscribed=1");
  resumeOnboardingPlan.mockRejectedValueOnce(new Error("Subscription is not active yet."));
  const { finishSetup } = renderHost(cardStep());
  expect(await screen.findByRole("alert")).toHaveTextContent("Subscription is not active yet.");
  expect(finishSetup).not.toHaveBeenCalled();
  expect(resumeOnboardingPlan).toHaveBeenCalledTimes(1);
  await userEvent.setup().click(screen.getByRole("button", { name: "Try again" }));
  await waitFor(() => expect(finishSetup).toHaveBeenCalledTimes(1));
});

it("keeps plan activation errors retryable", async () => {
  activateOnboardingPlan.mockRejectedValue(new Error("Could not select this plan."));
  renderHost(cardStep({ hasPaymentMethod: true }));
  await userEvent.setup().click(screen.getByTestId("first-run-card-continue"));
  expect(await screen.findByRole("alert")).toHaveTextContent("Could not select this plan.");
  expect(screen.getByTestId("first-run-card-continue")).toBeEnabled();
});

it("embeds the sandbox card form and verifies a trusted success before finishing", async () => {
  const { finishSetup } = renderHost(cardStep());
  await userEvent.setup().click(screen.getByRole("button", { name: "Continue with Pay-as-you-go" }));
  const frame = (await screen.findByTitle("Secure card setup")) as HTMLIFrameElement;
  expect(new URL(frame.src).origin).toBe("https://sandbox.polar.sh");
  expect(screen.queryByRole("radio")).not.toBeInTheDocument();
  const success = { type: "POLAR_PAYMENT_METHOD", event: "success", paymentMethodId: "pm_test" };
  await act(async () => {
    window.dispatchEvent(
      new MessageEvent("message", { data: success, origin: "https://evil.example", source: frame.contentWindow }),
    );
    window.dispatchEvent(
      new MessageEvent("message", { data: success, origin: "https://sandbox.polar.sh", source: window }),
    );
  });
  expect(resumeOnboardingPlan).not.toHaveBeenCalled();
  await act(async () => {
    window.dispatchEvent(
      new MessageEvent("message", { data: success, origin: "https://sandbox.polar.sh", source: frame.contentWindow }),
    );
  });
  await waitFor(() => expect(finishSetup).toHaveBeenCalledTimes(1));
  expect(resumeOnboardingPlan).toHaveBeenCalledWith("org-1", "pay_as_you_go", expect.any(String), false);
});

it("offers a new card session when the embedded form rejects an expired session", async () => {
  const { finishSetup } = renderHost(cardStep());
  await userEvent.setup().click(screen.getByRole("button", { name: "Continue with Pay-as-you-go" }));
  const frame = (await screen.findByTitle("Secure card setup")) as HTMLIFrameElement;
  await act(async () => {
    window.dispatchEvent(
      new MessageEvent("message", {
        origin: "https://sandbox.polar.sh",
        source: frame.contentWindow,
        data: { type: "POLAR_PAYMENT_METHOD", event: "error", code: "unauthorized" },
      }),
    );
  });
  expect(screen.getByRole("alert")).toHaveTextContent("Could not save your card");
  expect(finishSetup).not.toHaveBeenCalled();
  openCardSetup.mockResolvedValueOnce({ token: "polar_cst_refreshed", origin: "https://sandbox.polar.sh" });
  await userEvent.setup().click(screen.getByRole("button", { name: "Try again" }));
  await waitFor(() =>
    expect((screen.getByTitle("Secure card setup") as HTMLIFrameElement).src).toContain("polar_cst_refreshed"),
  );
});

it("only advertises signup credit when this organization has the full amount", () => {
  const balances = { welcomeCreditCents: 1000 };
  const { rerender, props } = renderHost(cardStep({ ...balances }));
  expect(screen.getByText("$10 in signup credit")).toBeInTheDocument();
  for (const amount of [undefined, 0, 999]) {
    const balance = { welcomeCreditCents: amount };
    rerender(
      <FirstRunCardScreenHost
        {...props}
        organizationId="another-org"
        flow={{ ...props.flow, cardStep: cardStep({ ...balance }) }}
      />,
    );
    expect(screen.queryByText("$10 in signup credit")).not.toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Starter" })).toBeEnabled();
  }
});
