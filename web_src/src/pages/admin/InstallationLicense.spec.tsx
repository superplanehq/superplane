import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import type { InstallationLicense as InstallationLicenseStatus } from "@/lib/license";
import InstallationLicense from "./InstallationLicense";

const { accountRef } = vi.hoisted(() => ({
  accountRef: { current: null as { license?: { hide_expiry_banner?: boolean } } | null },
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: accountRef.current,
    loading: false,
    setupRequired: false,
    refreshAccount: async () => undefined,
  }),
}));

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

beforeEach(() => {
  accountRef.current = null;
});

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
    expect(screen.getAllByText("(not included)", { exact: false })).toHaveLength(3);
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

  it("installs a license from a file", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) =>
      init?.method === "PUT" ? jsonResponse(enterpriseStatus) : jsonResponse(communityStatus),
    );
    vi.stubGlobal("fetch", fetchMock);

    renderPage();

    const file = new File(["  header.claims.sig\n"], "acme.license", { type: "text/plain" });
    fireEvent.change(await screen.findByTestId("license-file-input"), { target: { files: [file] } });

    expect(await screen.findByTestId("license-file-selected")).toHaveTextContent("acme.license");
    fireEvent.click(screen.getByTestId("license-install"));

    await waitFor(() => expect(screen.getByText("SuperPlane Enterprise")).toBeInTheDocument());
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

  it("shows that a revoked license no longer grants Enterprise features", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          ...enterpriseStatus,
          edition: "community",
          state: "revoked",
        }),
      ),
    );

    renderPage();

    await waitFor(() => expect(screen.getByTestId("license-state")).toHaveTextContent("Revoked"));
    expect(screen.getByText("This license was revoked. Enterprise features are not available.")).toBeInTheDocument();
    expect(screen.getByText("SuperPlane Community")).toBeInTheDocument();
  });

  it("shows the expiry badge for a revoked license unless the installation hides it", async () => {
    const expiresAt = new Date(Date.now() + 10 * 24 * 60 * 60 * 1000).toISOString();
    const revoked = {
      ...enterpriseStatus,
      edition: "community" as const,
      state: "revoked" as const,
      license: { ...enterpriseStatus.license!, expires_at: expiresAt },
    };

    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(revoked)),
    );
    const { unmount } = renderPage();

    expect(await screen.findByTestId("license-expiry-badge")).toHaveTextContent(/Expires in \d+ days/);

    unmount();
    accountRef.current = { license: { hide_expiry_banner: true } };
    renderPage();

    await screen.findByTestId("license-state");
    expect(screen.queryByTestId("license-expiry-badge")).not.toBeInTheDocument();
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

  it("installs without a dialog when the license removes no grants", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (init?.method === "PUT" && url.includes("dry_run=true")) {
        return jsonResponse({ event: "activation", removed_grants: [], auto_grant_organization_id: "" });
      }
      if (init?.method === "PUT") {
        return jsonResponse({
          ...enterpriseStatus,
          organization_count: 3,
          event: "activation",
        });
      }
      return jsonResponse(communityStatus);
    });
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    renderPage();

    fireEvent.change(await screen.findByTestId("license-key-input"), { target: { value: "header.claims.sig" } });
    await user.click(screen.getByTestId("license-install"));

    expect(await screen.findByTestId("license-install-note")).toHaveTextContent(
      "Enable Enterprise features for each organization.",
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("names the organization that receives every feature", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (init?.method === "PUT" && url.includes("dry_run=true")) {
          return jsonResponse({ event: "activation", removed_grants: [], auto_grant_organization_id: "" });
        }
        if (init?.method === "PUT") {
          return jsonResponse({
            ...enterpriseStatus,
            organization_count: 1,
            event: "activation",
            auto_grant_organization_id: "org-1",
            auto_grant_organization_name: "Acme",
          });
        }
        return jsonResponse(communityStatus);
      }),
    );

    renderPage();
    fireEvent.change(await screen.findByTestId("license-key-input"), { target: { value: "header.claims.sig" } });
    fireEvent.click(screen.getByTestId("license-install"));

    expect(await screen.findByTestId("license-install-note")).toHaveTextContent(
      "All Enterprise features are enabled for Acme.",
    );
  });

  it("links the organization count to the organizations that use a feature", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = input instanceof Request ? input.url : String(input);
        if (url.includes("/enterprise-features/groups/organizations")) {
          return jsonResponse({
            organizations: [{ organizationId: "org-1", name: "Acme", slug: "acme", source: "admin" }],
            total: "1",
          });
        }
        return jsonResponse({ ...enterpriseStatus, feature_counts: { groups: 2 } });
      }),
    );
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Used by 2 organizations" }));

    expect(await screen.findByText("Organizations that can use this feature.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Acme" })).toHaveAttribute("href", "/admin/organizations/org-1");
  });

  it("asks before an install that removes grants", async () => {
    const removed = [
      { organization_id: "org-1", organization_name: "Acme", feature_key: "groups" },
      { organization_id: "org-2", organization_name: "Beta", feature_key: "groups" },
    ];
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (init?.method === "PUT" && url.includes("dry_run=true")) {
        return jsonResponse({ event: "replacement", removed_grants: removed, auto_grant_organization_id: "" });
      }
      if (init?.method === "PUT") return jsonResponse(enterpriseStatus);
      return jsonResponse(communityStatus);
    });
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    renderPage();

    fireEvent.change(await screen.findByTestId("license-key-input"), { target: { value: "header.claims.sig" } });
    await user.click(screen.getByTestId("license-install"));

    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent(
      "This license does not include Groups. 2 organizations will lose access to these features.",
    );
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === "PUT")).toHaveLength(1);

    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(fetchMock.mock.calls.filter(([, init]) => init?.method === "PUT")).toHaveLength(1);
  });

  it("shows the organization count before removing the license", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (init?.method === "DELETE" && url.includes("dry_run=true")) {
        return jsonResponse({
          removed_grants: [
            { organization_id: "org-1", organization_name: "Acme", feature_key: "groups" },
            { organization_id: "org-1", organization_name: "Acme", feature_key: "custom_roles" },
            { organization_id: "org-2", organization_name: "Beta", feature_key: "groups" },
          ],
        });
      }
      if (init?.method === "DELETE") return jsonResponse(communityStatus);
      return jsonResponse(enterpriseStatus);
    });
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByTestId("license-remove"));
    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent(
      "Removing the license removes all Enterprise features from 2 organizations. You must enable them again after you install a new license.",
    );
    expect(
      fetchMock.mock.calls.filter(([url, init]) => init?.method === "DELETE" && !String(url).includes("dry_run")),
    ).toHaveLength(0);

    await user.click(within(dialog).getByRole("button", { name: "Remove license" }));
    await waitFor(() => expect(screen.getByText("SuperPlane Community")).toBeInTheDocument());
    expect(
      fetchMock.mock.calls.some((call) => call[1]?.method === "DELETE" && !String(call[0]).includes("dry_run")),
    ).toBe(true);
  });

  it("shows license activation history", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = input instanceof Request ? input.url : String(input);
        if (url.includes("/license/activations")) {
          return jsonResponse({
            activations: [
              {
                id: "9",
                licenseId: "abcdef12-1111-4111-8111-111111111111",
                licenseFeatures: ["groups"],
                activatedAt: "2026-03-01T12:00:00Z",
                organizationCount: 2,
                deactivationReason: "replaced",
                deactivatedAt: "2026-04-01T12:00:00Z",
                removedGrants: [{ organizationId: "org-1", organizationName: "Acme", featureKey: "custom_roles" }],
              },
            ],
          });
        }
        return jsonResponse(enterpriseStatus);
      }),
    );

    renderPage();

    const history = await screen.findByTestId("license-activations");
    expect(within(history).getByText("abcdef12")).toBeInTheDocument();
    expect(history).toHaveTextContent("Replaced");
    const user = userEvent.setup();
    await user.click(within(history).getByText("1 removed grant"));
    expect(within(history).getByText("Acme · Custom roles")).toBeInTheDocument();
  });
});
