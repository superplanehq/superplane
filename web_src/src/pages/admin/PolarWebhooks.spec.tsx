import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { PolarWebhooks } from "./PolarWebhooks";
import {
  POLAR_WEBHOOKS_EMPTY,
  POLAR_WEBHOOKS_HELP,
  POLAR_WEBHOOKS_NOT_CONFIGURED,
  POLAR_WEBHOOKS_REDELIVER,
  POLAR_WEBHOOKS_SENDING_AGAIN,
  POLAR_WEBHOOKS_TITLE,
  POLAR_WEBHOOKS_UNAUTHORIZED,
  POLAR_WEBHOOK_POLL_INTERVAL_MS,
  POLAR_WEBHOOK_REDELIVER_TIMEOUT_MS,
  formatPolarPayload,
  polarWebhookEventStatus,
  prunePendingPolarRedelivers,
  uniqueFailedEventIds,
  type PolarWebhookDelivery,
} from "./polarWebhookDeliveries";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

beforeAll(() => {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});

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

const duplicateFailedDelivery: PolarWebhookDelivery = {
  ...failedDelivery,
  id: "del_2",
  event_id: "evt_1",
};

const recoveredFailedDelivery: PolarWebhookDelivery = {
  ...failedDelivery,
  event_succeeded: true,
};

const otherFailedDelivery: PolarWebhookDelivery = {
  id: "del_3",
  created_at: "2026-09-14T12:02:00Z",
  succeeded: false,
  http_code: 500,
  response: "timeout",
  event_type: "subscription.updated",
  event_id: "evt_2",
  payload: "",
};

const succeededDelivery: PolarWebhookDelivery = {
  id: "del_ok",
  created_at: "2026-09-14T12:03:00Z",
  succeeded: true,
  http_code: 202,
  response: `{"status":"accepted"}`,
  event_type: "order.paid",
  event_id: "evt_ok",
  payload: "",
};

const jsonResponse = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/admin/polar-webhooks"]}>
      <PolarWebhooks />
    </MemoryRouter>,
  );

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.mocked(showErrorToast).mockClear();
  vi.mocked(showSuccessToast).mockClear();
});

describe("uniqueFailedEventIds", () => {
  it("keeps one id when Polar retried the same event", () => {
    expect(
      uniqueFailedEventIds([failedDelivery, duplicateFailedDelivery, succeededDelivery, otherFailedDelivery]),
    ).toEqual(["evt_1", "evt_2"]);
  });

  it("skips events Polar already delivered later", () => {
    expect(uniqueFailedEventIds([recoveredFailedDelivery, otherFailedDelivery])).toEqual(["evt_2"]);
  });
});

describe("polarWebhookEventStatus", () => {
  it("marks a pending event as sending again", () => {
    expect(polarWebhookEventStatus(failedDelivery, new Set(["evt_1"]))).toBe("sending");
  });

  it("marks a recovered failed delivery as succeeded", () => {
    expect(polarWebhookEventStatus(recoveredFailedDelivery, new Set())).toBe("succeeded");
  });

  it("marks an event without Polar success as failed", () => {
    expect(polarWebhookEventStatus(failedDelivery, new Set())).toBe("failed");
  });
});

describe("prunePendingPolarRedelivers", () => {
  const pending = new Map([
    ["evt_1", { startedAt: 1_000, knownDeliveryIds: ["del_1"] }],
  ]);

  it("keeps a pending event until Polar reports a later result", () => {
    const next = prunePendingPolarRedelivers(pending, [failedDelivery], 2_000);
    expect(next.has("evt_1")).toBe(true);
  });

  it("clears a pending event when Polar marks it succeeded", () => {
    const next = prunePendingPolarRedelivers(pending, [recoveredFailedDelivery], 2_000);
    expect(next.size).toBe(0);
  });

  it("clears a pending event when a newer delivery appears", () => {
    const next = prunePendingPolarRedelivers(
      pending,
      [
        failedDelivery,
        {
          ...failedDelivery,
          id: "del_retry",
          created_at: "2026-09-14T12:05:00Z",
        },
      ],
      2_000,
    );
    expect(next.size).toBe(0);
  });

  it("clears a pending event after the timeout", () => {
    const next = prunePendingPolarRedelivers(pending, [failedDelivery], 1_000 + POLAR_WEBHOOK_REDELIVER_TIMEOUT_MS);
    expect(next.size).toBe(0);
  });
});

describe("formatPolarPayload", () => {
  it("pretty-prints JSON payloads", () => {
    expect(formatPolarPayload(`{"type":"order.paid"}`)).toBe(`{\n  "type": "order.paid"\n}`);
  });

  it("keeps non-JSON payloads", () => {
    expect(formatPolarPayload("not json")).toBe("not json");
  });
});

describe("PolarWebhooks", () => {
  it("shows the Polar token empty state when billing is not configured", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          configured: false,
          items: [],
          total: 0,
          page: 1,
          limit: 50,
        }),
      ),
    );

    renderPage();

    expect(await screen.findByText(POLAR_WEBHOOKS_TITLE)).toBeInTheDocument();
    expect(screen.getByText(POLAR_WEBHOOKS_HELP)).toBeInTheDocument();
    expect(screen.getByText(POLAR_WEBHOOKS_NOT_CONFIGURED)).toBeInTheDocument();
  });

  it("lists failed deliveries and asks Polar to send an event again", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (init?.method === "POST" && url.includes("/admin/api/polar/webhooks/evt_1/redeliver")) {
        return jsonResponse({ status: "accepted" });
      }
      expect(url).toContain("succeeded=false");
      return jsonResponse({
        configured: true,
        items: [failedDelivery],
        total: 1,
        page: 1,
        limit: 50,
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("order.paid")).toBeInTheDocument();
    expect(screen.getAllByText("Failed").length).toBeGreaterThan(1);
    expect(screen.getByText("evt_1")).toBeInTheDocument();
    expect(screen.getByTestId("polar-webhooks-redeliver-failed")).toBeInTheDocument();
    expect(screen.getByTestId("polar-webhook-event-status")).toHaveTextContent("Failed");

    await user.click(screen.getByRole("button", { name: POLAR_WEBHOOKS_REDELIVER }));

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalledWith("Polar will send the event again.");
    });
    expect(fetchMock).toHaveBeenCalledWith("/admin/api/polar/webhooks/evt_1/redeliver", {
      method: "POST",
      credentials: "include",
    });
    expect(screen.getByTestId("polar-webhook-event-status")).toHaveTextContent(POLAR_WEBHOOKS_SENDING_AGAIN);
    expect(screen.getByRole("button", { name: POLAR_WEBHOOKS_SENDING_AGAIN })).toBeDisabled();
  });

  it("shows Event status Succeeded when Polar delivered the event later", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          configured: true,
          items: [recoveredFailedDelivery],
          total: 1,
          page: 1,
          limit: 50,
        }),
      ),
    );

    renderPage();

    expect(await screen.findByText("evt_1")).toBeInTheDocument();
    expect(screen.getByTestId("polar-webhook-event-status")).toHaveTextContent("Succeeded");
    expect(screen.queryByTestId("polar-webhooks-redeliver-failed")).not.toBeInTheDocument();
  });

  it("keeps Sending again until Polar marks the event succeeded", async () => {
    vi.useFakeTimers({ toFake: ["setInterval"] });
    let eventSucceeded = false;
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "POST") {
        return jsonResponse({ status: "accepted" });
      }
      return jsonResponse({
        configured: true,
        items: [{ ...failedDelivery, event_succeeded: eventSucceeded }],
        total: 1,
        page: 1,
        limit: 50,
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("evt_1")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: POLAR_WEBHOOKS_REDELIVER }));

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalledWith("Polar will send the event again.");
    });
    expect(screen.getByTestId("polar-webhook-event-status")).toHaveTextContent(POLAR_WEBHOOKS_SENDING_AGAIN);

    eventSucceeded = true;
    await act(async () => {
      vi.advanceTimersByTime(POLAR_WEBHOOK_POLL_INTERVAL_MS);
    });

    await waitFor(() => {
      expect(screen.getByTestId("polar-webhook-event-status")).toHaveTextContent("Succeeded");
    });
  });

  it("shows Polar response and payload when a delivery is expanded", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          configured: true,
          items: [failedDelivery],
          total: 1,
          page: 1,
          limit: 50,
        }),
      ),
    );

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("evt_1")).toBeInTheDocument();
    expect(screen.queryByText("Polar response")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Show delivery details" }));

    expect(screen.getByText("Polar response")).toBeInTheDocument();
    expect(screen.getByText("unable to apply order")).toBeInTheDocument();
    expect(screen.getByText(/"type": "order.paid"/)).toBeInTheDocument();
  });

  it("redelivers unique failed events on the page", async () => {
    const posted: string[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (init?.method === "POST") {
          posted.push(url);
          return jsonResponse({ status: "accepted" });
        }
        return jsonResponse({
          configured: true,
          items: [failedDelivery, duplicateFailedDelivery, otherFailedDelivery],
          total: 3,
          page: 1,
          limit: 50,
        });
      }),
    );

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByTestId("polar-webhooks-redeliver-failed")).toBeInTheDocument();
    await user.click(screen.getByTestId("polar-webhooks-redeliver-failed"));

    await waitFor(() => {
      expect(posted).toEqual([
        "/admin/api/polar/webhooks/evt_1/redeliver",
        "/admin/api/polar/webhooks/evt_2/redeliver",
      ]);
    });
    expect(showSuccessToast).toHaveBeenCalledWith("Polar will send 2 events again.");
  });

  it("does not bulk redeliver events Polar already delivered later", async () => {
    const posted: string[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = String(input);
        if (init?.method === "POST") {
          posted.push(url);
          return jsonResponse({ status: "accepted" });
        }
        return jsonResponse({
          configured: true,
          items: [recoveredFailedDelivery, otherFailedDelivery],
          total: 2,
          page: 1,
          limit: 50,
        });
      }),
    );

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByTestId("polar-webhooks-redeliver-failed")).toBeInTheDocument();
    await user.click(screen.getByTestId("polar-webhooks-redeliver-failed"));

    await waitFor(() => {
      expect(posted).toEqual(["/admin/api/polar/webhooks/evt_2/redeliver"]);
    });
    expect(showSuccessToast).toHaveBeenCalledWith("Polar will send 1 event again.");
  });

  it("keeps the newest rows when a slow load finishes after a filter change", async () => {
    let finishSucceededLoad: ((value: Response) => void) | undefined;
    const fetchMock = vi.fn((input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("succeeded=true")) {
        return new Promise<Response>((resolve) => {
          finishSucceededLoad = resolve;
        });
      }
      const items = url.includes("succeeded=false") ? [failedDelivery] : [otherFailedDelivery];
      return Promise.resolve(jsonResponse({ configured: true, items, total: 1, page: 1, limit: 50 }));
    });
    vi.stubGlobal("fetch", fetchMock);

    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText("evt_1")).toBeInTheDocument();

    await user.click(screen.getByTestId("polar-webhooks-status"));
    await user.click(screen.getByRole("option", { name: "Succeeded" }));
    await waitFor(() => {
      expect(finishSucceededLoad).toBeDefined();
    });

    await user.click(screen.getByTestId("polar-webhooks-status"));
    await user.click(screen.getByRole("option", { name: "All" }));
    expect(await screen.findByText("evt_2")).toBeInTheDocument();

    finishSucceededLoad?.(jsonResponse({ configured: true, items: [succeededDelivery], total: 1, page: 1, limit: 50 }));
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(screen.queryByText("evt_ok")).not.toBeInTheDocument();
    expect(screen.getByText("evt_2")).toBeInTheDocument();
  });

  it("shows an empty filter state", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          configured: true,
          items: [],
          total: 0,
          page: 1,
          limit: 50,
        }),
      ),
    );

    renderPage();

    expect(await screen.findByText(POLAR_WEBHOOKS_EMPTY)).toBeInTheDocument();
    expect(screen.queryByTestId("polar-webhooks-redeliver-failed")).not.toBeInTheDocument();
  });

  it("shows Polar unauthorized copy when the token lacks webhook scopes", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response(POLAR_WEBHOOKS_UNAUTHORIZED, { status: 502 })),
    );

    renderPage();

    expect(await screen.findByText(POLAR_WEBHOOKS_UNAUTHORIZED)).toBeInTheDocument();
    expect(showErrorToast).toHaveBeenCalledWith(POLAR_WEBHOOKS_UNAUTHORIZED);
  });
});
