import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { BillingPlans } from "./BillingPlans";

afterEach(() => vi.unstubAllGlobals());

it("keeps entered values and shows failed saves inside the editor", async () => {
  const saves: unknown[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input) === "/admin/api/polar/products?kind=subscription") {
        return new Response(JSON.stringify([{ id: "prod_team", name: "Team subscription" }]));
      }
      if (init?.method === "POST") {
        saves.push(JSON.parse(String(init.body)));
        return new Response("Polar seat price must match the catalog price.", { status: 400 });
      }
      return new Response("[]");
    }),
  );
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await waitFor(() => expect(screen.getByRole("button", { name: "Create plan" })).toBeEnabled());
  await user.click(screen.getByRole("button", { name: "Create plan" }));
  await user.type(screen.getByLabelText("Name", { exact: true }), "Team");
  await user.type(screen.getByLabelText("Key", { exact: true }), "team");
  await user.click(screen.getByRole("combobox", { name: "Polar product" }));
  await user.click(await screen.findByRole("option", { name: /Team subscription/ }));
  const seatLimit = screen.getByLabelText("Seat limit", { exact: true });
  await user.type(seatLimit, "25");
  await user.click(screen.getByRole("switch", { name: "Per seat" }));
  const maximum = screen.getByLabelText("Maximum seats", { exact: true });
  expect(maximum).toHaveValue(25);
  for (const invalid of ["", "0", "-1", "1001", "1.5"]) {
    fireEvent.change(maximum, { target: { value: invalid } });
    fireEvent.submit(screen.getByRole("button", { name: "Save plan" }).closest("form")!);
    expect(within(screen.getByRole("dialog")).getByRole("alert")).toHaveTextContent(
      "Maximum seats must be an integer from 1 to 1,000.",
    );
    expect(saves).toHaveLength(0);
  }
  fireEvent.change(maximum, { target: { value: "80" } });
  await user.click(screen.getByRole("button", { name: "Save plan" }));
  const dialog = within(screen.getByRole("dialog"));
  expect(await dialog.findByRole("alert")).toHaveTextContent("Polar seat price must match the catalog price.");
  expect(dialog.getByLabelText("Name", { exact: true })).toHaveValue("Team");
  expect(dialog.getByLabelText("Key", { exact: true })).toHaveValue("team");
  expect(dialog.getByRole("combobox", { name: "Polar product" })).toHaveTextContent("Team subscription");
  expect(dialog.getByLabelText("Maximum seats", { exact: true })).toHaveValue(80);
  expect(saves).toEqual([expect.objectContaining({ seat_limit: 80, per_seat: true })]);
  await user.click(dialog.getByRole("button", { name: "Cancel" }));
  await user.click(screen.getByRole("button", { name: "Create plan" }));
  expect(within(screen.getByRole("dialog")).queryByRole("alert")).not.toBeInTheDocument();
}, 15000);

it("preserves unlimited seat limits for plans without per-seat pricing", async () => {
  const saves: unknown[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "POST") saves.push(JSON.parse(String(init.body)));
      return new Response("[]");
    }),
  );
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <BillingPlans />
    </MemoryRouter>,
  );
  await waitFor(() => expect(screen.getByRole("button", { name: "Create plan" })).toBeEnabled());
  await user.click(screen.getByRole("button", { name: "Create plan" }));
  await user.type(screen.getByLabelText("Name", { exact: true }), "Flat");
  await user.type(screen.getByLabelText("Key", { exact: true }), "flat");
  await user.type(screen.getByLabelText("Seat limit", { exact: true }), "2000");
  await user.clear(screen.getByLabelText("Seat limit", { exact: true }));
  await user.click(screen.getByRole("button", { name: "Save plan" }));
  await waitFor(() => expect(saves).toEqual([expect.objectContaining({ seat_limit: null, per_seat: false })]));
});
