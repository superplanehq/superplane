import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { MemoryRouter } from "react-router";
import { RegistrationStats, type RegistrationReport } from "./RegistrationStats";

const report: RegistrationReport = {
  total: 6,
  trial_granted_cents: 6000,
  trial_spent_cents: 2350,
  plan_activations: { team: 3, starter: 5, pay_as_you_go: 12 },
  plan_activation_tracking_started_at: "2026-01-01T00:00:00Z",
  items: [
    { id: "1", name: "Acme", owner_email: "ada@acme.example", plan: "Trial", available_credit_cents: 750 },
    {
      id: "2",
      name: "Northstar",
      owner_email: "owner@northstar.example",
      plan: "Business",
      available_credit_cents: 5200,
    },
    { id: "3", name: "Orbit", owner_email: "owner@orbit.example", plan: "Trial", available_credit_cents: 1000 },
    { id: "4", name: "Fieldwork", owner_email: "owner@fieldwork.example", plan: "None", available_credit_cents: 0 },
    {
      id: "5",
      name: "An organization with a long name",
      owner_email: "a.long.owner.email@organization.example",
      plan: "Trial",
      available_credit_cents: 700,
    },
    { id: "6", name: "Tandem", owner_email: "owner@tandem.example", plan: "Business", available_credit_cents: 4500 },
  ],
};

const meta = {
  title: "Admin/Registration statistics",
  component: RegistrationStats,
  decorators: [
    (Story) => (
      <MemoryRouter>
        <div className="bg-slate-100 p-6 dark:bg-gray-950">
          <Story />
        </div>
      </MemoryRouter>
    ),
  ],
  parameters: {
    msw: {
      handlers: [
        http.get("/admin/api/stats", ({ request }) => {
          const offset = Number(new URL(request.url).searchParams.get("offset") ?? 0);
          return HttpResponse.json({ ...report, items: report.items.slice(offset, offset + 5) });
        }),
      ],
    },
  },
} satisfies Meta<typeof RegistrationStats>;
export default meta;
type Story = StoryObj<typeof meta>;

// Local noon today. The default period (today) starts before this time.
const trackingStartedToday = new Date();
trackingStartedToday.setHours(12, 0, 0, 0);

function statsHandler(body: Partial<RegistrationReport>) {
  return {
    msw: {
      handlers: [http.get("/admin/api/stats", () => HttpResponse.json({ ...report, ...body }))],
    },
  };
}

export const Populated: Story = {};
export const Empty: Story = {
  parameters: statsHandler({
    items: [],
    total: 0,
    trial_granted_cents: 0,
    trial_spent_cents: 0,
    plan_activations: { team: 0, starter: 0, pay_as_you_go: 0 },
  }),
};
export const ZeroActivations: Story = {
  parameters: statsHandler({ plan_activations: { team: 0, starter: 0, pay_as_you_go: 0 } }),
};
export const EmptyRegistrationsWithActivations: Story = {
  parameters: statsHandler({
    items: [],
    total: 0,
    trial_granted_cents: 0,
    trial_spent_cents: 0,
    plan_activations: { team: 2, starter: 1, pay_as_you_go: 4 },
  }),
};
export const CrossesTrackingCutoff: Story = {
  parameters: statsHandler({ plan_activation_tracking_started_at: trackingStartedToday.toISOString() }),
};
export const Loading: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get("/admin/api/stats", async () => {
          await delay("infinite");
          return HttpResponse.json(report);
        }),
      ],
    },
  },
};
export const Error: Story = {
  parameters: { msw: { handlers: [http.get("/admin/api/stats", () => new HttpResponse(null, { status: 500 }))] } },
};
