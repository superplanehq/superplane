import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { SentryWebhooks } from "./SentryWebhooks";
import {
  SENTRY_WEBHOOKS_EMPTY,
  SENTRY_WEBHOOKS_HELP,
  SENTRY_WEBHOOKS_PAGE_EMPTY,
  SENTRY_WEBHOOKS_TITLE,
} from "./sentryWebhookReceipts";

const renderPage = () =>
  render(
    <MemoryRouter>
      <SentryWebhooks />
    </MemoryRouter>,
  );

describe("SentryWebhooks", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows an empty list", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ items: [], total: 0, page: 1, limit: 50 }),
      }),
    );

    renderPage();

    expect(await screen.findByText(SENTRY_WEBHOOKS_TITLE)).toBeInTheDocument();
    expect(screen.getByText(SENTRY_WEBHOOKS_HELP)).toBeInTheDocument();
    expect(await screen.findByText(SENTRY_WEBHOOKS_EMPTY)).toBeInTheDocument();
  });

  it("shows a receipt without a payload", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          items: [
            {
              id: "receipt-1",
              received_at: "2026-09-29T11:14:00Z",
              hook_resource: "issue",
              action: "created",
              installation_uuid: "install-1",
              organization_slug: "acme",
              project_slug: "javascript-react-f",
              issue_id: "99",
              issue_short_id: "JS-9",
              http_status: 200,
              outcome: "accepted",
              integration_count: 1,
              task_ids: ["6f1c2a40-1b2e-4c3d-9a8b-0e1f2a3b4c5d"],
            },
          ],
          total: 1,
          page: 1,
          limit: 50,
        }),
      }),
    );

    renderPage();

    expect(await screen.findByText("javascript-react-f")).toBeInTheDocument();
    expect(screen.getByText("JS-9")).toBeInTheDocument();
    expect(screen.getByText("Accepted")).toBeInTheDocument();
    expect(screen.getByText("created")).toBeInTheDocument();
    expect(screen.getByText("install-1")).toBeInTheDocument();
    expect(screen.getByText("6f1c2a40-1b2e-4c3d-9a8b-0e1f2a3b4c5d")).toBeInTheDocument();
  });

  it("keeps Previous when a later page is empty", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockImplementation(async (input: string) => {
        if (input.includes("page=2")) {
          return {
            ok: true,
            json: async () => ({ items: [], total: 50, page: 2, limit: 50 }),
          };
        }
        return {
          ok: true,
          json: async () => ({
            items: [
              {
                id: "receipt-1",
                received_at: "2026-09-29T11:14:00Z",
                hook_resource: "issue",
                action: "created",
                installation_uuid: "install-1",
                organization_slug: "acme",
                project_slug: "javascript-react-f",
                issue_id: "99",
                issue_short_id: "JS-9",
                http_status: 200,
                outcome: "accepted",
                integration_count: 1,
              },
            ],
            total: 51,
            page: 1,
            limit: 50,
          }),
        };
      }),
    );

    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "Next" }));
    expect(await screen.findByText(SENTRY_WEBHOOKS_PAGE_EMPTY)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Previous" }));
    expect(await screen.findByText("javascript-react-f")).toBeInTheDocument();
  });
});
