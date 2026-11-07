import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { FactoryWebhookWarnings } from "./FactoryWebhookWarnings";

const warning = (id: string, automationId: string) => ({
  id,
  repository: "acme/app",
  eventTypes: ["issues"],
  affectedAutomations: [{ id: automationId, name: automationId }],
  state: "failed",
  message: "GitHub could not register the webhook for issues. Check the repository settings, then try again.",
});

vi.mock("@/hooks/useFactoryWebhookData", () => ({
  useFactoryWebhookData: () => ({
    warnings: [warning("hook-a", "automation-a"), warning("hook-b", "automation-b")],
    retry: vi.fn(),
    retrying: () => false,
    retryTimedOut: () => false,
    refresh: vi.fn(),
  }),
}));

describe("FactoryWebhookWarnings", () => {
  it("shows each workspace warning once and limits detail warnings to affected automations", () => {
    const { rerender } = render(<FactoryWebhookWarnings organizationId="org" factoryId="factory" canRetry />);
    expect(screen.getAllByRole("alert")).toHaveLength(2);

    rerender(<FactoryWebhookWarnings organizationId="org" factoryId="factory" automationId="automation-a" canRetry />);
    expect(screen.getAllByRole("alert")).toHaveLength(1);
    expect(screen.getByRole("alert")).toHaveTextContent("automation-a");
  });
});
