import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { RegistrationStats } from "./RegistrationStats";

afterEach(() => vi.unstubAllGlobals());

const activationsBeforeTracking = { team: 0, starter: 0, pay_as_you_go: 0 };
const trackedSince = "2020-01-01T00:00:00.000Z";

function renderStats() {
  render(
    <MemoryRouter>
      <RegistrationStats />
    </MemoryRouter>,
  );
}

function metricValue(label: string) {
  return screen.getByText(label).nextElementSibling?.textContent;
}

function stubStats(body: (offset: number) => Record<string, unknown>) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: string) => {
      const url = new URL(input, "http://localhost");
      return new Response(JSON.stringify(body(Number(url.searchParams.get("offset")))));
    }),
  );
}

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
          plan_activations: { team: 2, starter: 1, pay_as_you_go: 0 },
          plan_activation_tracking_started_at: trackedSince,
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
  renderStats();
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
  expect(metricValue("Team activations")).toBe("2");
  await user.click(screen.getByRole("combobox", { name: "Period" }));
  await user.click(screen.getByRole("option", { name: "Custom" }));
  fireEvent.change(screen.getByLabelText("Start date"), { target: { value: "2026-03-01" } });
  fireEvent.change(screen.getByLabelText("End date"), { target: { value: "2026-03-02" } });
  fail = true;
  await user.click(screen.getByRole("button", { name: "Apply dates" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Failed to load statistics");
  expect(requests.at(-1)?.searchParams.get("offset")).toBe("0");
  const start = new Date(requests.at(-1)!.searchParams.get("start")!);
  const end = new Date(requests.at(-1)!.searchParams.get("end")!);
  expect(start.getDate()).toBe(1);
  expect(end.getDate()).toBe(3);
  fail = false;
  await user.click(screen.getByRole("button", { name: "Try again" }));
  expect(await screen.findByRole("link", { name: "Organization 0" })).toBeInTheDocument();
  await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
});

it("shows zero activation counts and the tracking start", async () => {
  stubStats(() => ({
    total: 0,
    trial_granted_cents: 0,
    trial_spent_cents: 0,
    items: [],
    plan_activations: activationsBeforeTracking,
    plan_activation_tracking_started_at: trackedSince,
  }));
  renderStats();
  expect(await screen.findByText("Team activations")).toBeInTheDocument();
  expect(metricValue("Team activations")).toBe("0");
  expect(metricValue("Starter activations")).toBe("0");
  expect(metricValue("Pay-as-you-go activations")).toBe("0");
  expect(screen.getByText(/Tracking started on/)).toHaveTextContent(new Date(trackedSince).toLocaleDateString());
  expect(screen.queryByRole("note")).not.toBeInTheDocument();
});

it("shows activations when no organization registered in the period", async () => {
  stubStats(() => ({
    total: 0,
    trial_granted_cents: 0,
    trial_spent_cents: 0,
    items: [],
    plan_activations: { team: 2, starter: 0, pay_as_you_go: 4 },
    plan_activation_tracking_started_at: trackedSince,
  }));
  renderStats();
  expect(await screen.findByText("Team activations")).toBeInTheDocument();
  expect(metricValue("Team activations")).toBe("2");
  expect(metricValue("Pay-as-you-go activations")).toBe("4");
  expect(
    screen.getByText("No organizations registered in this period. Choose a different date range."),
  ).toBeInTheDocument();
});

it("warns when the selected period starts before activation tracking began", async () => {
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  stubStats(() => ({
    total: 0,
    trial_granted_cents: 0,
    trial_spent_cents: 0,
    items: [],
    plan_activations: activationsBeforeTracking,
    plan_activation_tracking_started_at: tomorrow.toISOString(),
  }));
  renderStats();
  expect(await screen.findByRole("note")).toHaveTextContent("Activations before");
  expect(screen.getByRole("note")).toHaveTextContent("are not tracked");
});
