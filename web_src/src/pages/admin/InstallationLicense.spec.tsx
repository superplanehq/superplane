import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import type { InstallationLicense as InstallationLicenseStatus } from "@/lib/license";
import InstallationLicense from "./InstallationLicense";

const communityStatus: InstallationLicenseStatus = {
  edition: "community",
  state: "none",
  source: "database",
  managed_by_configuration: false,
};

const enterpriseStatus: InstallationLicenseStatus = {
  edition: "enterprise",
  state: "active",
  source: "database",
  managed_by_configuration: false,
  license: {
    id: "11111111-1111-4111-8111-111111111111",
    customer_id: "22222222-2222-4222-8222-222222222222",
    features: ["groups"],
    issued_at: "2026-01-01T00:00:00Z",
    valid_from: "2026-01-01T00:00:00Z",
    expires_at: "2099-01-01T00:00:00Z",
  },
};

const jsonResponse = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/admin/license"]}>
      <InstallationLicense />
    </MemoryRouter>,
  );

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("InstallationLicense", () => {
  it("shows Community with locked features before a license is installed", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(communityStatus)),
    );

    renderPage();

    expect(await screen.findByText("SuperPlane Community")).toBeInTheDocument();
    expect(screen.getByTestId("license-state")).toHaveTextContent("No license");
    expect(screen.getAllByText("(not included)", { exact: false })).toHaveLength(2);
    expect(screen.getByTestId("license-install")).toBeDisabled();
    expect(screen.queryByTestId("license-remove")).not.toBeInTheDocument();
  });

  it("installs a license and shows the granted features", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) =>
      init?.method === "PUT" ? jsonResponse(enterpriseStatus) : jsonResponse(communityStatus),
    );
    vi.stubGlobal("fetch", fetchMock);

    renderPage();

    fireEvent.change(await screen.findByTestId("license-key-input"), { target: { value: "  header.claims.sig  " } });
    fireEvent.click(screen.getByTestId("license-install"));

    expect(await screen.findByText("SuperPlane Enterprise")).toBeInTheDocument();
    expect(screen.getByTestId("license-state")).toHaveTextContent("Active");
    expect(screen.getByTestId("license-remove")).toBeInTheDocument();

    const putCall = fetchMock.mock.calls.find(([, init]) => init?.method === "PUT");
    expect(JSON.parse(String(putCall?.[1]?.body))).toEqual({ license: "header.claims.sig" });
  });

  it("shows the server error when the license is rejected", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) =>
        init?.method === "PUT"
          ? new Response("The license signature is not valid.", { status: 422 })
          : jsonResponse(communityStatus),
      ),
    );

    renderPage();

    fireEvent.change(await screen.findByTestId("license-key-input"), { target: { value: "bad" } });
    fireEvent.click(screen.getByTestId("license-install"));

    expect(await screen.findByRole("alert")).toHaveTextContent("The license signature is not valid.");
    expect(screen.getByText("SuperPlane Community")).toBeInTheDocument();
  });

  it("does not offer changes when the configuration manages the license", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ ...enterpriseStatus, source: "file", managed_by_configuration: true })),
    );

    renderPage();

    expect(await screen.findByText("Managed license")).toBeInTheDocument();
    expect(screen.queryByTestId("license-key-input")).not.toBeInTheDocument();
  });

  it("explains why a configured license is invalid", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ ...communityStatus, state: "invalid", reason: "unknown_key", source: "file" })),
    );

    renderPage();

    await waitFor(() => expect(screen.getByTestId("license-state")).toHaveTextContent("Invalid"));
    expect(screen.getByText(/does not trust the key/)).toBeInTheDocument();
  });

  it("uploads a key list when automatic key updates are off", async () => {
    const withKeys = (version: number) => ({
      ...communityStatus,
      trusted_keys: { state: version > 1 ? "synced" : "disabled", version },
    });
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) =>
      init?.method === "PUT" ? jsonResponse(withKeys(2)) : jsonResponse(withKeys(1)),
    );
    vi.stubGlobal("fetch", fetchMock);

    renderPage();

    expect(await screen.findByTestId("license-trusted-keys-state")).toHaveTextContent("Automatic updates are off");
    fireEvent.change(screen.getByTestId("license-key-list-input"), { target: { value: " a.b.c " } });
    fireEvent.click(screen.getByTestId("license-key-list-upload"));

    await waitFor(() => expect(screen.getByTestId("license-trusted-keys-state")).toHaveTextContent("Up to date"));
    expect(screen.queryByTestId("license-key-list-input")).not.toBeInTheDocument();

    const putCall = fetchMock.mock.calls.find(([, init]) => init?.method === "PUT");
    expect(String(putCall?.[0])).toBe("/admin/api/installation/license/keys");
    expect(JSON.parse(String(putCall?.[1]?.body))).toEqual({ key_list: "a.b.c" });
  });
});
