import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { OrganizationSuspensionControl } from "./OrganizationSuspensionControl";
import type { OrganizationOverview } from "./OrgOverviewPanel";

const organization: OrganizationOverview = {
  id: "org-1",
  name: "Demo",
  slug: "demo",
  description: "",
  canvas_count: 1,
  task_count: 0,
  done_task_count: 0,
  member_count: 1,
  workspace_count: 0,
  repository_count: 0,
};

afterEach(() => vi.unstubAllGlobals());

it("requires a reason and confirms suspension before sending the admin request", async () => {
  const fetchMock = vi.fn(async () => new Response("{}", { status: 200 }));
  vi.stubGlobal("fetch", fetchMock);
  const onChange = vi.fn();
  render(<OrganizationSuspensionControl organization={organization} onChange={onChange} />);
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "Suspend organization" }));
  const buttons = screen.getAllByRole("button", { name: "Suspend organization" });
  const confirm = buttons[buttons.length - 1];
  expect(confirm).toBeDisabled();
  expect(fetchMock).not.toHaveBeenCalled();
  await user.type(screen.getByLabelText("Suspension reason"), "Unattended automation");
  await user.click(confirm);
  expect(fetchMock).toHaveBeenCalledWith(
    "/admin/api/organizations/org-1/suspend",
    expect.objectContaining({
      method: "POST",
      body: JSON.stringify({ reason: "Unattended automation" }),
    }),
  );
  expect(onChange).toHaveBeenCalledTimes(1);
});

it("keeps the dialog open when cancellation still blocks reactivation", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response("Work is still stopping.", { status: 409 })),
  );
  const onChange = vi.fn();
  render(
    <OrganizationSuspensionControl
      organization={{ ...organization, suspended_at: new Date().toISOString() }}
      onChange={onChange}
    />,
  );
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "Reactivate organization" }));
  const buttons = screen.getAllByRole("button", { name: "Reactivate organization" });
  await user.click(buttons[buttons.length - 1]);
  expect(screen.getByRole("dialog")).toBeInTheDocument();
  expect(screen.getAllByRole("alert").some((alert) => alert.textContent?.includes("Work is still stopping."))).toBe(
    true,
  );
  expect(onChange).not.toHaveBeenCalled();
});
