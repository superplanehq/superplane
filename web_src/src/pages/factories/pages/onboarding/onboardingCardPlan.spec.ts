import { beforeEach, expect, it, vi } from "bun:test";

import { activateOnboardingPlan } from "./onboardingCardPlan";

const selectPlan = vi.fn();
vi.mock("@/api-client", () => ({ organizationsSelectBillingPlan: selectPlan }));

beforeEach(() => {
  selectPlan.mockReset();
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
