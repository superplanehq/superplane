import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { InstallationEncryption } from "./InstallationEncryption";

describe("InstallationEncryption", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("allows a new migration after a failed job", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const body = String(input).endsWith("/migrations")
          ? {
              migrations: [
                {
                  id: "job-1",
                  targetKeyId: "key-1",
                  state: "failed",
                  storageLocation: "secrets.data",
                  errorMessage: "Could not decrypt a value.",
                },
              ],
            }
          : { keys: [{ id: "key-1", masterKeyRef: "local-v1" }], activeKeyId: "key-1" };
        return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
      }),
    );

    render(<InstallationEncryption />);

    expect(await screen.findByText("Could not decrypt a value.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create data key" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Start migration" })).toBeEnabled();
  });

  it("creates a data key through the keys endpoint", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const body = String(input).endsWith("/migrations")
        ? { migrations: [] }
        : init?.method === "POST"
          ? { activeKeyId: "key-2" }
          : { keys: [{ id: "key-1", masterKeyRef: "local-v1" }], activeKeyId: "key-1" };
      return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    render(<InstallationEncryption />);

    await user.click(await screen.findByRole("button", { name: "Create data key" }));
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Create data key" }));

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        "/admin/api/installation/keys",
        expect.objectContaining({ method: "POST" }),
      ),
    );
  });

  it("keeps key controls available when migrations are unavailable", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        if (String(input).endsWith("/migrations")) {
          return new Response("Migrations are unavailable while legacy writes are enabled.", { status: 400 });
        }
        return new Response(
          JSON.stringify({ keys: [{ id: "key-1", masterKeyRef: "local-v1" }], activeKeyId: "key-1" }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }),
    );

    render(<InstallationEncryption />);

    expect(await screen.findByText("key-1")).toBeInTheDocument();
    expect(screen.getByText("Migrations are unavailable while legacy writes are enabled.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create data key" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Rewrap data keys" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Start migration" })).toBeDisabled();
  });
});
