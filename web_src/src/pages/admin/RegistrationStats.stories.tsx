import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { MemoryRouter } from "react-router";
import { RegistrationStats, type RegistrationReport } from "./RegistrationStats";

const report: RegistrationReport = {
  total: 6,
  trial_granted_cents: 6000,
  trial_spent_cents: 2350,
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
    { id: "6", name: "Tandem", owner_email: "test@superplane.com", plan: "Business", available_credit_cents: 4500 },
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
          const params = new URL(request.url).searchParams;
          const offset = Number(params.get("offset") ?? 0);
          const excludeInternal = params.get("exclude_internal") === "true";
          const items = excludeInternal ? report.items.slice(0, 5) : report.items;
          return HttpResponse.json({
            total: items.length,
            ...(offset
              ? {}
              : {
                  trial_granted_cents: excludeInternal ? 5000 : report.trial_granted_cents,
                  trial_spent_cents: excludeInternal ? 2000 : report.trial_spent_cents,
                }),
            items: items.slice(offset, offset + 5),
          });
        }),
      ],
    },
  },
} satisfies Meta<typeof RegistrationStats>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Populated: Story = {};
export const Empty: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get("/admin/api/stats", () =>
          HttpResponse.json({ items: [], total: 0, trial_granted_cents: 0, trial_spent_cents: 0 }),
        ),
      ],
    },
  },
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
