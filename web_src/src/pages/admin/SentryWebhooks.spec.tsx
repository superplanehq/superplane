import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { DatadogWebhooks } from "./DatadogWebhooks";
import { LinearWebhooks } from "./LinearWebhooks";
import { DATADOG_WEBHOOKS_HELP } from "./datadogWebhookReceipts";
import { LINEAR_WEBHOOKS_HELP } from "./linearWebhookReceipts";
import { SentryWebhooks } from "./SentryWebhooks";
import {
  SENTRY_WEBHOOKS_EMPTY,
  SENTRY_WEBHOOKS_HELP,
  SENTRY_WEBHOOKS_LOAD_ERROR,
  SENTRY_WEBHOOKS_PAGE_EMPTY,
  SENTRY_WEBHOOKS_PROJECT_EMPTY,
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

    expect(await screen.findByText(SENTRY_WEBHOOKS_HELP)).toBeInTheDocument();
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

  it("filters by project and returns to the first page", async () => {
    const fetchMock = vi.fn().mockImplementation(async (input: string) => {
      if (input.includes("project=production")) {
        return {
          ok: true,
          json: async () => ({ items: [], total: 0, page: 1, limit: 50 }),
        };
      }
      if (input.includes("page=2")) {
        return {
          ok: true,
          json: async () => ({
            items: [sentryReceipt("page-2")],
            total: 51,
            page: 2,
            limit: 50,
          }),
        };
      }
      return {
        ok: true,
        json: async () => ({
          items: [sentryReceipt("page-1")],
          total: 51,
          page: 1,
          limit: 50,
        }),
      };
    });
    vi.stubGlobal("fetch", fetchMock);

    renderPage();

    expect(await screen.findByTestId("sentry-webhooks-project")).toBeInTheDocument();
    fireEvent.click(await screen.findByRole("button", { name: "Next" }));
    expect(await screen.findByText("page-2")).toBeInTheDocument();

    fireEvent.change(screen.getByTestId("sentry-webhooks-project"), { target: { value: " production " } });

    expect(screen.queryByText("page-2")).not.toBeInTheDocument();
    expect(await screen.findByText(SENTRY_WEBHOOKS_PROJECT_EMPTY)).toBeInTheDocument();
    expect(screen.getByTestId("sentry-webhooks-project")).toHaveValue(" production ");
    expect(screen.queryByText(SENTRY_WEBHOOKS_EMPTY)).not.toBeInTheDocument();

    const projectRequests = fetchMock.mock.calls
      .map(([input]) => String(input))
      .filter((url) => url.includes("project=production"));
    expect(projectRequests).toHaveLength(1);
    expect(projectRequests[0]).toContain("page=1");
    expect(projectRequests[0]).not.toContain("page=2");
  });

  it("keeps the project field when the list fails to load", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
      }),
    );

    renderPage();

    expect(await screen.findByText(SENTRY_WEBHOOKS_LOAD_ERROR)).toBeInTheDocument();
    expect(screen.getByTestId("sentry-webhooks-project")).toBeInTheDocument();
    expect(screen.getByText("Project")).toBeInTheDocument();
  });

  it("does not show a project field for Datadog or Linear", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ items: [], total: 0, page: 1, limit: 50 }),
      }),
    );

    const datadog = render(
      <MemoryRouter>
        <DatadogWebhooks />
      </MemoryRouter>,
    );
    expect(await screen.findByText(DATADOG_WEBHOOKS_HELP)).toBeInTheDocument();
    expect(screen.queryByTestId("sentry-webhooks-project")).not.toBeInTheDocument();
    datadog.unmount();

    render(
      <MemoryRouter>
        <LinearWebhooks />
      </MemoryRouter>,
    );
    expect(await screen.findByText(LINEAR_WEBHOOKS_HELP)).toBeInTheDocument();
    expect(screen.queryByTestId("sentry-webhooks-project")).not.toBeInTheDocument();
  });
});

function sentryReceipt(projectSlug: string) {
  return {
    id: projectSlug,
    received_at: "2026-09-29T11:14:00Z",
    hook_resource: "issue",
    action: "created",
    installation_uuid: "install-1",
    organization_slug: "acme",
    project_slug: projectSlug,
    issue_id: "99",
    issue_short_id: "JS-9",
    http_status: 200,
    outcome: "accepted",
    integration_count: 1,
  };
}
