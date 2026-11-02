import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { delay, http, HttpResponse } from "msw";
import { MemoryRouter, Route, Routes } from "react-router";
import { expect, userEvent, within } from "storybook/test";
import { DeleteOrganizationControl } from "./DeleteOrganizationControl";

const endpoint = "/admin/api/organizations/org-1";
const meta = {
  title: "Admin/DeleteOrganizationControl",
  component: DeleteOrganizationControl,
  args: { organization: { id: "org-1", name: "Acme" } },
  parameters: {
    layout: "centered",
    msw: { handlers: [http.delete(endpoint, () => HttpResponse.json({ status: "deleted" }))] },
  },
  decorators: [
    (Story) => (
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter initialEntries={["/admin/organizations/org-1"]}>
          <Routes>
            <Route path="/admin/organizations/:orgId" element={<Story />} />
            <Route path="/admin" element={<p role="status">Organization deleted. All organizations.</p>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    ),
  ],
} satisfies Meta<typeof DeleteOrganizationControl>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Confirmation: Story = {
  play: async ({ canvasElement }) => {
    await userEvent.click(within(canvasElement).getByRole("button", { name: "Delete organization" }));
  },
};

export const PendingDelete: Story = {
  parameters: {
    msw: {
      handlers: [
        http.delete(endpoint, async () => {
          await delay("infinite");
          return HttpResponse.json({ status: "deleted" });
        }),
      ],
    },
  },
  play: async ({ canvasElement }) => {
    await userEvent.click(within(canvasElement).getByRole("button", { name: "Delete organization" }));
    const dialog = within(await within(document.body).findByRole("dialog"));
    await userEvent.type(dialog.getByLabelText("Organization name"), "Acme");
    await userEvent.click(dialog.getByRole("button", { name: "Delete organization" }));
    await expect(dialog.getByRole("button", { name: "Deleting..." })).toBeDisabled();
  },
};

export const BillingRejection: Story = {
  parameters: {
    msw: {
      handlers: [
        http.delete(endpoint, () =>
          HttpResponse.json(
            {
              code: 10,
              message:
                "Cancel the subscription in Billing before you delete the organization. You can delete it after the paid period ends.",
            },
            { status: 409 },
          ),
        ),
      ],
    },
  },
  play: async ({ canvasElement }) => {
    await userEvent.click(within(canvasElement).getByRole("button", { name: "Delete organization" }));
    const dialog = within(await within(document.body).findByRole("dialog"));
    await userEvent.type(dialog.getByLabelText("Organization name"), "Acme");
    await userEvent.click(dialog.getByRole("button", { name: "Delete organization" }));
    await expect(await dialog.findByRole("alert")).toHaveTextContent("Cancel the subscription in Billing");
    await expect(dialog.getByRole("button", { name: "Keep organization" })).toBeEnabled();
  },
};
