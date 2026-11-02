import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "bun:test";
import { client } from "@/api-client/client.gen";
import { FleetEnabledButton } from "./FleetEnabledButton";

const originalConfig = client.getConfig();
afterEach(() => {
  cleanup();
  client.setConfig(originalConfig);
});

for (const enabled of [true, false]) {
  it(`${enabled ? "disables" : "enables"} the fleet without replacing its specification`, async () => {
    let body: unknown;
    client.setConfig({
      baseUrl: "http://localhost",
      fetch: async (input) => {
        const request = input as Request;
        expect(new URL(request.url).pathname).toBe("/api/v1/organizations/org-1/fleets/local");
        expect(request.method).toBe("PATCH");
        body = await request.json();
        return Response.json({ fleet: { id: "local", enabled: !enabled } });
      },
    });
    const refresh = vi.fn();
    render(<FleetEnabledButton organizationId="org-1" fleet={{ id: "local", enabled }} refresh={refresh} />);
    fireEvent.click(screen.getByRole("button", { name: enabled ? "Disable fleet" : "Enable fleet" }));
    if (enabled) fireEvent.click(screen.getByRole("button", { name: "Confirm disable" }));
    await waitFor(() => expect(refresh).toHaveBeenCalledTimes(1));
    expect(body).toEqual({ enabled: !enabled });
  });
}

it("keeps the fleet unchanged and offers retry after a failed update", async () => {
  let failed = true;
  client.setConfig({
    baseUrl: "http://localhost",
    fetch: async () =>
      failed ? Response.json({ code: 14 }, { status: 503 }) : Response.json({ fleet: { id: "local", enabled: false } }),
  });
  const refresh = vi.fn();
  render(<FleetEnabledButton organizationId="org-1" fleet={{ id: "local", enabled: true }} refresh={refresh} />);
  fireEvent.click(screen.getByRole("button", { name: "Disable fleet" }));
  fireEvent.click(screen.getByRole("button", { name: "Confirm disable" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Could not disable the fleet. Check access and retry.");
  expect(refresh).not.toHaveBeenCalled();
  failed = false;
  fireEvent.click(screen.getByRole("button", { name: "Confirm disable" }));
  await waitFor(() => expect(refresh).toHaveBeenCalledTimes(1));
  expect(screen.queryByRole("alert")).not.toBeInTheDocument();
});
