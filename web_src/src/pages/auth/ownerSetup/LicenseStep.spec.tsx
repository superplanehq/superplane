import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { LicenseStep } from "./LicenseStep";

const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });

const communityStatus = { edition: "community", state: "none", source: "database", managed_by_configuration: false };

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("LicenseStep", () => {
  it("lets the owner skip the step", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(communityStatus)),
    );
    const onContinue = vi.fn();

    render(<LicenseStep onContinue={onContinue} />);

    fireEvent.click(await screen.findByTestId("owner-setup-license-skip"));
    expect(onContinue).toHaveBeenCalledTimes(1);
  });

  it("continues immediately when the configuration manages the license", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ ...communityStatus, source: "file", managed_by_configuration: true })),
    );
    const onContinue = vi.fn();

    render(<LicenseStep onContinue={onContinue} />);

    await waitFor(() => expect(onContinue).toHaveBeenCalledTimes(1));
    expect(screen.queryByTestId("owner-setup-license")).not.toBeInTheDocument();
  });

  it("continues when the license status cannot be loaded", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("", { status: 500 })),
    );
    const onContinue = vi.fn();

    render(<LicenseStep onContinue={onContinue} />);

    await waitFor(() => expect(onContinue).toHaveBeenCalledTimes(1));
  });

  it("shows a loading state until the license keys are ready", async () => {
    const states = ["syncing", "synced"];
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({ ...communityStatus, trusted_keys: { state: states.shift() ?? "synced", version: 1 } }),
      ),
    );

    render(<LicenseStep onContinue={vi.fn()} />);

    expect(screen.getByTestId("owner-setup-license-loading")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByTestId("owner-setup-license")).toBeInTheDocument(), { timeout: 3000 });
  });

  it("confirms activation after a license is installed", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) =>
        init?.method === "PUT"
          ? jsonResponse({
              edition: "enterprise",
              state: "active",
              source: "database",
              managed_by_configuration: false,
              license: {
                id: "1",
                customer_id: "2",
                features: ["custom_roles", "groups"],
                issued_at: "2026-01-01T00:00:00Z",
                valid_from: "2026-01-01T00:00:00Z",
                expires_at: "2099-01-01T00:00:00Z",
              },
            })
          : jsonResponse(communityStatus),
      ),
    );
    const onContinue = vi.fn();

    render(<LicenseStep onContinue={onContinue} />);

    fireEvent.change(await screen.findByTestId("license-key-input"), { target: { value: "a.b.c" } });
    fireEvent.click(screen.getByTestId("license-install"));

    expect(await screen.findByTestId("owner-setup-license-active")).toHaveTextContent("Custom roles, Groups");
    fireEvent.click(screen.getByText("Continue"));
    expect(onContinue).toHaveBeenCalledTimes(1);
  });
});
