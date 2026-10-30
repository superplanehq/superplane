import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { RegistrationStats } from "./RegistrationStats";

afterEach(() => vi.unstubAllGlobals());

it("paginates registrations, resets the page for a date filter, and retries failures", async () => {
  const requests: URL[] = [];
  let fail = false;
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: string) => {
      const url = new URL(input, "http://localhost");
      requests.push(url);
      if (fail) return new Response("Unavailable", { status: 500 });
      const offset = Number(url.searchParams.get("offset"));
      return new Response(
        JSON.stringify({
          total: 6,
          ...(offset ? {} : { trial_granted_cents: 6000, trial_spent_cents: 1200 }),
          items: Array.from({ length: offset ? 1 : 5 }, (_, i) => ({
            id: `org-${offset + i}`,
            name: `Organization ${offset + i}`,
            owner_email: "owner@example.com",
            plan: "Trial",
            available_credit_cents: 800,
          })),
        }),
      );
    }),
  );
  const user = userEvent.setup();
  render(
    <MemoryRouter>
      <RegistrationStats />
    </MemoryRouter>,
  );
  expect(await screen.findByRole("link", { name: "Organization 0" })).toHaveAttribute(
    "href",
    "/admin/organizations/org-0",
  );
  expect(screen.getAllByRole("row")).toHaveLength(6);
  expect(screen.getByText("$60.00")).toBeInTheDocument();
  expect(screen.getByText("$12.00")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Next" }));
  expect(await screen.findByRole("link", { name: "Organization 5" })).toBeInTheDocument();
  expect(requests.at(-1)?.searchParams.get("offset")).toBe("5");
  expect(screen.getByText("$60.00")).toBeInTheDocument();
  expect(screen.getByText("$12.00")).toBeInTheDocument();
  expect(requests.at(-1)?.searchParams.get("exclude_internal")).toBe("true");
  await user.click(screen.getByRole("checkbox", { name: "Exclude @superplane.com owners" }));
  await screen.findByRole("link", { name: "Organization 0" });
  expect(requests.at(-1)?.searchParams.get("exclude_internal")).toBe("false");
  expect(requests.at(-1)?.searchParams.get("offset")).toBe("0");
  await user.click(screen.getByRole("combobox", { name: "Registration period" }));
  await user.click(screen.getByRole("option", { name: "Custom" }));
  fireEvent.change(screen.getByLabelText("Start date"), { target: { value: "2026-03-01" } });
  fireEvent.change(screen.getByLabelText("End date"), { target: { value: "2026-03-02" } });
  fail = true;
  await user.click(screen.getByRole("button", { name: "Apply dates" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Failed to load statistics");
  expect(requests.at(-1)?.searchParams.get("offset")).toBe("0");
  expect(requests.at(-1)?.searchParams.get("exclude_internal")).toBe("false");
  const start = new Date(requests.at(-1)!.searchParams.get("start")!);
  const end = new Date(requests.at(-1)!.searchParams.get("end")!);
  expect(start.getDate()).toBe(1);
  expect(end.getDate()).toBe(3);
  fail = false;
  await user.click(screen.getByRole("button", { name: "Try again" }));
  expect(await screen.findByRole("link", { name: "Organization 0" })).toBeInTheDocument();
  await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
});
