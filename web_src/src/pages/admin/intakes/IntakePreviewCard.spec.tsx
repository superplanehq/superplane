import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { ADD_INTAKE_COPY } from "@/pages/factories/pages/lineIntakeModel";

import type { AdminIntakeEntry } from "./intakeCatalogModel";
import { IntakePreviewCard } from "./IntakePreviewCard";

const ACME_ID = "6f1f8a2e-3a43-4a8e-9a55-0d6b2d0c8f11";

const datadog: AdminIntakeEntry = {
  key: "datadog",
  name: "Datadog errors",
  category: "error_tracking",
  status: "beta",
  status_note: "",
  enabled_for_all: false,
  implemented: true,
  deletable: false,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  updated_by_name: "",
  organizations: [{ id: ACME_ID, name: "Acme", added_at: new Date().toISOString() }],
};

function stubPreviewFetch() {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(String(input), "http://localhost");
      if (url.pathname !== "/admin/api/intake-catalog/datadog/preview") {
        return new Response("not found", { status: 404 });
      }
      const added = url.searchParams.get("organization_id") === ACME_ID;
      const body = { key: "datadog", status: "beta", organization_added: added, available: added };
      return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
    }),
  );
}

function renderPreview(entry: AdminIntakeEntry) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
  return render(<IntakePreviewCard entry={entry} />, { wrapper });
}

describe("IntakePreviewCard", () => {
  beforeEach(stubPreviewFetch);
  afterEach(() => vi.unstubAllGlobals());

  it("shows Coming soon to a company without access and Beta to a company that the admin added", async () => {
    const user = userEvent.setup();
    renderPreview(datadog);

    const addIntake = await screen.findByTestId("intake-preview-addIntake");
    expect(within(addIntake).getByTestId("add-intake-template-datadog")).toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
    expect(screen.getByTestId("intake-preview-onboardingTickets")).toHaveTextContent(
      "Not shown in Onboarding: tickets. The code does not support this intake here.",
    );

    await user.click(screen.getByRole("combobox", { name: "View as" }));
    await user.click(screen.getByRole("option", { name: "Acme" }));

    const card = await within(screen.getByTestId("intake-preview-addIntake")).findByTestId("add-intake-beta-badge");
    expect(card).toHaveTextContent(ADD_INTAKE_COPY.beta);
    expect(screen.getByTestId("add-intake-template-datadog")).not.toHaveTextContent(ADD_INTAKE_COPY.comingSoon);
  });
});
