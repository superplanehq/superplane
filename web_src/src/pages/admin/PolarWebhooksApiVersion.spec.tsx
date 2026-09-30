import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { PolarWebhooks } from "./PolarWebhooks";
import {
  isPolarApiVersionMismatch,
  mismatchedPolarWebhookEndpoints,
  POLAR_WEBHOOKS_VERSION_CHECK_FAILED,
  polarWebhookEndpointVersionWarning,
  type PolarWebhookDelivery,
  type PolarWebhookEndpoint,
} from "./polarWebhookDeliveries";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

const failedDelivery: PolarWebhookDelivery = {
  id: "del_1",
  created_at: "2026-09-14T12:00:00Z",
  succeeded: false,
  http_code: 500,
  response: "unable to apply order",
  event_type: "order.paid",
  event_id: "evt_1",
  payload: `{"type":"order.paid"}`,
};

const superPlaneEndpoint: PolarWebhookEndpoint = {
  id: "end_superplane",
  url: "https://app.superplane.com/api/v1/polar/webhooks",
  api_version: "2026-04",
  format: "raw",
  current: true,
};

const otherAppEndpoint: PolarWebhookEndpoint = {
  id: "end_other",
  url: "https://other.example.com/hooks/polar",
  api_version: "2026-10",
  format: "raw",
  current: false,
};

const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });

function stubPolarAdminApi(endpoints: PolarWebhookEndpoint[], deliveries: PolarWebhookDelivery[]) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).endsWith("/admin/api/polar/webhooks/endpoints")) {
        return jsonResponse({ configured: true, api_version: "2026-10", endpoints });
      }
      return jsonResponse({ configured: true, items: deliveries, total: deliveries.length, page: 1, limit: 50 });
    }),
  );
}

async function renderLoadedPage() {
  render(
    <MemoryRouter initialEntries={["/admin/polar-webhooks"]}>
      <PolarWebhooks />
    </MemoryRouter>,
  );
  expect(await screen.findByText("evt_1")).toBeInTheDocument();
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("mismatchedPolarWebhookEndpoints", () => {
  it("returns this installation's endpoints that use another API version", () => {
    expect(
      mismatchedPolarWebhookEndpoints({
        configured: true,
        api_version: "2026-10",
        endpoints: [
          superPlaneEndpoint,
          { ...superPlaneEndpoint, id: "end_hooks", url: "https://hooks.superplane.com/api/v1/polar/webhooks" },
          { ...superPlaneEndpoint, id: "end_pinned", api_version: "2026-10" },
          { ...otherAppEndpoint, api_version: "2026-04" },
        ],
      }).map((endpoint) => endpoint.id),
    ).toEqual(["end_superplane", "end_hooks"]);
  });

  it("returns nothing when Polar is not configured or the response is missing", () => {
    expect(mismatchedPolarWebhookEndpoints(null)).toEqual([]);
    expect(
      mismatchedPolarWebhookEndpoints({ configured: false, api_version: "2026-10", endpoints: [superPlaneEndpoint] }),
    ).toEqual([]);
  });
});

describe("isPolarApiVersionMismatch", () => {
  it("flags only a known version that differs from the pinned one", () => {
    expect(isPolarApiVersionMismatch("2026-04", "2026-10")).toBe(true);
    expect(isPolarApiVersionMismatch("2026-10", "2026-10")).toBe(false);
    expect(isPolarApiVersionMismatch(undefined, "2026-10")).toBe(false);
    expect(isPolarApiVersionMismatch("2026-04", "")).toBe(false);
  });
});

describe("PolarWebhooks API version", () => {
  it("warns when the SuperPlane webhook endpoint uses another API version", async () => {
    stubPolarAdminApi([superPlaneEndpoint, { ...otherAppEndpoint, api_version: "2026-04" }], [failedDelivery]);

    await renderLoadedPage();

    const warning = screen.getByTestId("polar-webhook-version-warning");
    expect(warning).toHaveTextContent(polarWebhookEndpointVersionWarning(superPlaneEndpoint, "2026-10"));
    expect(warning).not.toHaveTextContent(otherAppEndpoint.url);
  });

  it("does not warn when the webhook endpoint uses the pinned API version", async () => {
    stubPolarAdminApi([{ ...superPlaneEndpoint, api_version: "2026-10" }], [failedDelivery]);

    await renderLoadedPage();

    expect(screen.queryByTestId("polar-webhook-version-warning")).not.toBeInTheDocument();
  });

  it("shows that the version check failed when Polar endpoints do not load", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        if (String(input).endsWith("/admin/api/polar/webhooks/endpoints")) {
          return new Response("Failed to load Polar webhook endpoints", { status: 502 });
        }
        return jsonResponse({ configured: true, items: [failedDelivery], total: 1, page: 1, limit: 50 });
      }),
    );

    await renderLoadedPage();

    expect(await screen.findByTestId("polar-webhook-version-check-failed")).toHaveTextContent(
      POLAR_WEBHOOKS_VERSION_CHECK_FAILED,
    );
    expect(screen.queryByTestId("polar-webhook-version-warning")).not.toBeInTheDocument();
  });

  it("shows the event API version when a delivery is expanded", async () => {
    stubPolarAdminApi([], [{ ...failedDelivery, api_version: "2026-04" }]);
    const user = userEvent.setup();

    await renderLoadedPage();
    await user.click(screen.getByRole("button", { name: "Show delivery attempts" }));

    expect(screen.getByTestId("polar-webhook-api-version")).toHaveTextContent(
      "API version: 2026-04 (SuperPlane uses 2026-10)",
    );
  });
});
