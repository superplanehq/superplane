import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { InstallationEncryption } from "./InstallationEncryption";

describe("InstallationEncryption", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

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

  it("continues polling after a temporary migration request failure", async () => {
    vi.useFakeTimers();
    let migrationRequests = 0;
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        if (String(input).endsWith("/migrations")) {
          migrationRequests += 1;
          if (migrationRequests === 2) throw new Error("Network unavailable");
          return new Response(
            JSON.stringify({
              migrations: [
                { id: "job-1", targetKeyId: "key-1", state: migrationRequests === 1 ? "running" : "completed" },
              ],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }
        return new Response(
          JSON.stringify({ keys: [{ id: "key-1", masterKeyRef: "local-v1" }], activeKeyId: "key-1" }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        );
      }),
    );

    render(<InstallationEncryption />);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });
    expect(screen.getByText("running")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(screen.getByText("job-1")).toBeInTheDocument();
    expect(screen.getByText("Could not load migrations.")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(migrationRequests).toBe(3);
    expect(screen.getByText("completed")).toBeInTheDocument();
  });
});
