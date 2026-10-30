import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it, vi } from "bun:test";
import type { OrganizationsBillingPlanOption } from "@/api-client";
import { BillingSeatCheckout } from "./BillingSeatCheckout";

const plan: OrganizationsBillingPlanOption = {
  key: "team",
  name: "Team",
  perSeat: true,
  priceCents: "2000",
  seatMaximum: 10,
};

function checkout(pending: number) {
  return <BillingSeatCheckout plan={plan} occupied={1} pending={pending} disabled={false} onSubscribe={vi.fn()} />;
}

it("keeps the selected seat count when billing data refreshes", async () => {
  const user = userEvent.setup();
  const { rerender } = render(checkout(2));
  const seats = screen.getByRole("spinbutton", { name: "Team seats" });
  expect(seats).toHaveValue(3);
  await user.click(screen.getByRole("button", { name: "Decrease Team seats" }));
  expect(seats).toHaveValue(2);

  rerender(checkout(6));
  expect(screen.getByRole("spinbutton", { name: "Team seats" })).toHaveValue(2);
});
