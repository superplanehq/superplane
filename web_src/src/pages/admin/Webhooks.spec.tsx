import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { DATADOG_WEBHOOKS_HELP } from "./datadogWebhookReceipts";
import { SENTRY_WEBHOOKS_EMPTY, SENTRY_WEBHOOKS_HELP } from "./sentryWebhookReceipts";
import { Webhooks } from "./Webhooks";

const renderPage = (path = "/admin/webhooks") =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Webhooks />
    </MemoryRouter>,
  );

describe("Webhooks", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows Sentry calls on the Sentry tab", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ items: [], total: 0, page: 1, limit: 50 }),
      }),
    );

    renderPage();

    expect(screen.getByRole("heading", { name: "Webhooks" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Sentry" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Datadog" })).toBeInTheDocument();
    expect(await screen.findByText(SENTRY_WEBHOOKS_HELP)).toBeInTheDocument();
    expect(await screen.findByText(SENTRY_WEBHOOKS_EMPTY)).toBeInTheDocument();
    expect(screen.queryByText(DATADOG_WEBHOOKS_HELP)).not.toBeInTheDocument();
  });

  it("shows Datadog calls on the Datadog tab", async () => {
    const fetchMock = vi.fn().mockImplementation(async (input: string) => {
      if (input.includes("/admin/api/datadog/webhooks")) {
        return {
          ok: true,
          json: async () => ({
            items: [
              {
                id: "receipt-1",
                received_at: "2026-09-29T11:14:00Z",
                integration_id: "integration-1",
                organization_id: "org-1",
                event_type: "error_tracking_alert",
                alert_transition: "Triggered",
                alert_id: "867",
                service: "checkout",
                issue_id: "11111111-1111-4111-8111-111111111111",
                http_status: 200,
                outcome: "accepted",
                subscription_count: 1,
                task_ids: ["6f1c2a40-1b2e-4c3d-9a8b-0e1f2a3b4c5d"],
              },
            ],
            total: 1,
            page: 1,
            limit: 50,
          }),
        };
      }
      return {
        ok: true,
        json: async () => ({ items: [], total: 0, page: 1, limit: 50 }),
      };
    });
    vi.stubGlobal("fetch", fetchMock);

    renderPage("/admin/webhooks?service=datadog");

    expect(await screen.findByText(DATADOG_WEBHOOKS_HELP)).toBeInTheDocument();
    expect(await screen.findByText("checkout")).toBeInTheDocument();
    expect(screen.getByText("Accepted")).toBeInTheDocument();
    expect(screen.getByText("867")).toBeInTheDocument();
    expect(screen.getByText("6f1c2a40-1b2e-4c3d-9a8b-0e1f2a3b4c5d")).toBeInTheDocument();
    expect(screen.queryByText("payload-secret-body")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("tab", { name: "Sentry" }));
    expect(await screen.findByText(SENTRY_WEBHOOKS_EMPTY)).toBeInTheDocument();
  });
});
