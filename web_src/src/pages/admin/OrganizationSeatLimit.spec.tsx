import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { OrganizationSeatLimit } from "./OrganizationSeatLimit";

afterEach(() => vi.unstubAllGlobals());
it("loads, saves custom and unlimited overrides, and restores the plan state", async () => {
  const requests: { method: string; body?: unknown }[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      const method = init?.method || "GET";
      const body = init?.body ? JSON.parse(String(init.body)) : undefined;
      requests.push({ method, body });
      return new Response(
        JSON.stringify({
          plan_limit: 10,
          override: method === "PUT" ? body : null,
          effective_limit: method === "PUT" ? body.seat_limit : 10,
        }),
      );
    }),
  );
  const user = userEvent.setup();
  render(<OrganizationSeatLimit orgId="org-1" />);
  expect(await screen.findByText("Plan limit: 10 · Effective limit: 10")).toBeInTheDocument();
  await user.click(screen.getByRole("combobox"));
  await user.click(screen.getByRole("option", { name: "Custom limit" }));
  fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "3" } });
  await user.click(screen.getByRole("button", { name: "Save seat limit" }));
  expect(await screen.findByText("Plan limit: 10 · Effective limit: 3")).toBeInTheDocument();
  await user.click(screen.getByRole("combobox"));
  await user.click(screen.getByRole("option", { name: "Unlimited" }));
  await user.click(screen.getByRole("button", { name: "Save seat limit" }));
  expect(await screen.findByText("Plan limit: 10 · Effective limit: Unlimited")).toBeInTheDocument();
  await user.click(screen.getByRole("combobox"));
  await user.click(screen.getByRole("option", { name: "Use plan limit" }));
  await user.click(screen.getByRole("button", { name: "Save seat limit" }));
  await waitFor(() =>
    expect(requests).toEqual([
      { method: "GET", body: undefined },
      { method: "PUT", body: { seat_limit: 3 } },
      { method: "PUT", body: { seat_limit: null } },
      { method: "DELETE", body: undefined },
    ]),
  );
  expect(await screen.findByText("Plan limit: 10 · Effective limit: 10")).toBeInTheDocument();
});
it("shows loading and a recoverable request error", async () => {
  let reject!: (reason: Error) => void;
  vi.stubGlobal(
    "fetch",
    vi.fn(
      () =>
        new Promise<Response>((_resolve, fail) => {
          reject = fail;
        }),
    ),
  );
  render(<OrganizationSeatLimit orgId="org-1" />);
  expect(screen.getByText("Loading seat limit...")).toBeInTheDocument();
  await act(async () => reject(new Error("offline")));
  expect(await screen.findByRole("alert")).toHaveTextContent("Failed to load seat limit.");
  expect(screen.getByRole("button", { name: "Try again" })).toBeEnabled();
});

it("validates custom capacity and preserves the saved state when saving fails", async () => {
  let rejectSave!: (reason: Error) => void;
  const fetchMock = vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
    if (init?.method === "GET")
      return Promise.resolve(new Response(JSON.stringify({ plan_limit: 10, override: null, effective_limit: 10 })));
    return new Promise<Response>((_resolve, reject) => {
      rejectSave = reject;
    });
  });
  vi.stubGlobal("fetch", fetchMock);
  const user = userEvent.setup();
  render(<OrganizationSeatLimit orgId="org-1" />);
  await screen.findByRole("button", { name: "Save seat limit" });
  await user.click(screen.getByRole("combobox"));
  await user.click(screen.getByRole("option", { name: "Custom limit" }));
  fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "0" } });
  await user.click(screen.getByRole("button", { name: "Save seat limit" }));
  expect(screen.getByRole("alert")).toHaveTextContent("Enter a whole number");
  expect(fetchMock).toHaveBeenCalledTimes(1);
  fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "3" } });
  await user.click(screen.getByRole("button", { name: "Save seat limit" }));
  expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
  await act(async () => rejectSave(new Error("offline")));
  expect(await screen.findByRole("alert")).toHaveTextContent("Failed to save seat limit.");
  expect(screen.getByText("Plan limit: 10 · Effective limit: 10")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Save seat limit" })).toBeEnabled();
});
