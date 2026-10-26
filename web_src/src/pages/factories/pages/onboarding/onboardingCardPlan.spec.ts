import { beforeEach, expect, it, vi } from "bun:test";

import { activateOnboardingPlan, createOnboardingCardSession, resumeOnboardingPlan } from "./onboardingCardPlan";

const selectPlan = vi.fn();
const createSession = vi.fn();
const syncBilling = vi.fn();
vi.mock("@/api-client", () => ({
  organizationsSelectBillingPlan: selectPlan,
  organizationsCreatePaymentMethodSession: createSession,
  organizationsSyncOrganizationBilling: syncBilling,
}));

beforeEach(() => {
  selectPlan.mockReset();
  createSession.mockReset();
  syncBilling.mockReset();
});

it("gets a card session for the correct Polar environment without navigating", async () => {
  createSession.mockResolvedValue({
    data: { portalUrl: "https://sandbox.polar.sh/portal", sessionToken: "polar_cst_test" },
  });
  const assign = vi.spyOn(window.location, "assign");
  try {
    await expect(createOnboardingCardSession("org-1")).resolves.toEqual({
      token: "polar_cst_test",
      origin: "https://sandbox.polar.sh",
    });
    expect(assign).not.toHaveBeenCalled();
    createSession.mockResolvedValue({ data: { portalUrl: "https://evil.example", sessionToken: "token" } });
    await expect(createOnboardingCardSession("org-1")).rejects.toThrow("Could not open card setup");
    createSession.mockResolvedValue({ data: { portalUrl: "https://polar.sh/portal" } });
    await expect(createOnboardingCardSession("org-1")).rejects.toThrow("Could not open card setup");
  } finally {
    assign.mockRestore();
  }
});

it("verifies the saved card before activating Pay-as-you-go", async () => {
  syncBilling.mockResolvedValueOnce({ data: { planCategory: "trial", hasPaymentMethod: false } });
  await expect(resumeOnboardingPlan("org-1", "pay_as_you_go", "/setup", false)).rejects.toThrow("No card was saved");
  expect(selectPlan).not.toHaveBeenCalled();
  syncBilling.mockResolvedValueOnce({ data: { planCategory: "trial", hasPaymentMethod: true } });
  selectPlan.mockResolvedValue({ data: {} });
  await expect(resumeOnboardingPlan("org-1", "pay_as_you_go", "/setup", false)).resolves.toBe(false);
  expect(selectPlan).toHaveBeenCalledWith(
    expect.objectContaining({ body: { planKey: "pay_as_you_go", returnPath: "/setup" } }),
  );
});

it("requires a confirmed subscription on checkout return without starting another checkout", async () => {
  syncBilling.mockResolvedValueOnce({ data: { planCategory: "trial", hasPaymentMethod: true } });
  await expect(resumeOnboardingPlan("org-1", "starter", "/setup", true)).rejects.toThrow("not active yet");
  syncBilling.mockResolvedValueOnce({
    data: { planCategory: "paid", catalogKey: "starter", polarSubscriptionStatus: "incomplete" },
  });
  await expect(resumeOnboardingPlan("org-1", "starter", "/setup", true)).rejects.toThrow("not active yet");
  syncBilling.mockResolvedValueOnce({
    data: { planCategory: "paid", catalogKey: "starter", polarSubscriptionStatus: "active" },
  });
  await expect(resumeOnboardingPlan("org-1", "starter", "/setup", true)).resolves.toBe(false);
  expect(selectPlan).not.toHaveBeenCalled();
});

it("activates a plan that needs no checkout", async () => {
  selectPlan.mockResolvedValue({ data: {} });
  await expect(activateOnboardingPlan("org-1", "payg")).resolves.toBe(false);
  expect(selectPlan).toHaveBeenCalledWith(expect.objectContaining({ body: { planKey: "payg" } }));
});

it("leaves for checkout when the plan needs it", async () => {
  selectPlan.mockResolvedValue({ data: { checkoutUrl: "https://polar.example/checkout" } });
  const previousAssign = window.location.assign.bind(window.location);
  const assign = vi.fn();
  window.location.assign = assign;
  try {
    await expect(
      activateOnboardingPlan("org-1", "team", { seats: 1, returnPath: "/org-1/setup?step=card" }),
    ).resolves.toBe(true);
    expect(assign).toHaveBeenCalledWith("https://polar.example/checkout");
    expect(selectPlan).toHaveBeenCalledWith(
      expect.objectContaining({
        body: { planKey: "team", seats: 1, returnPath: "/org-1/setup?step=card" },
      }),
    );
  } finally {
    window.location.assign = previousAssign;
  }
});
