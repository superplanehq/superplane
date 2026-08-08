import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { OrganizationsIntegration } from "@/api-client";
import { IntegrationCatalogRow, type IntegrationCatalogEntry } from "./IntegrationCatalogRow";

vi.mock("@/ui/componentSidebar/integrationIcons", () => ({
  IntegrationIcon: ({ integrationName }: { integrationName?: string }) => (
    <span data-testid={`icon-${integrationName ?? "unknown"}`} />
  ),
}));

function instance(state: string | undefined, name = "cursor"): OrganizationsIntegration {
  return {
    metadata: { id: `id-${name}-${state ?? "none"}`, name, integrationName: "cursor" },
    status: state ? { state } : {},
  };
}

function entry(overrides: Partial<IntegrationCatalogEntry> = {}): IntegrationCatalogEntry {
  return {
    providerName: "cursor",
    providerLabel: "Cursor",
    integrationDef: { name: "cursor", label: "Cursor", description: "Build workflows with Cursor" },
    instances: [],
    ...overrides,
  };
}

function renderRow(overrides: Partial<IntegrationCatalogEntry> = {}, props: Record<string, unknown> = {}) {
  return render(
    <ul>
      <IntegrationCatalogRow
        entry={entry(overrides)}
        isActive={false}
        canCreateIntegrations
        canUpdateIntegrations
        permissionsLoading={false}
        onConnect={vi.fn()}
        onConfigure={vi.fn()}
        rowRef={vi.fn()}
        {...props}
      />
    </ul>,
  );
}

describe("IntegrationCatalogRow", () => {
  it("renders the provider with its connect action", () => {
    renderRow();

    expect(screen.getByRole("heading", { name: "Cursor" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Connect" })).toBeInTheDocument();
  });

  it("marks the row as current when highlighted", () => {
    renderRow({}, { isActive: true });

    expect(screen.getByRole("listitem")).toHaveAttribute("aria-current", "true");
  });

  it("shows a single status badge per connected instance", () => {
    renderRow({ instances: [instance("ready")] });

    expect(screen.getByText("1 connected instance")).toBeInTheDocument();
    expect(screen.getByText("Ready")).toBeInTheDocument();
    expect(screen.getByText("cursor")).toBeInTheDocument();
  });

  it.each([
    ["ready", "Ready"],
    ["error", "Error"],
    ["pending", "Pending"],
    [undefined, "Unknown"],
  ])("labels the %s state as %s", (state, expected) => {
    renderRow({ instances: [instance(state)] });

    expect(screen.getByText(expected)).toBeInTheDocument();
  });

  it("pluralises the connected instance count", () => {
    renderRow({ instances: [instance("ready", "one"), instance("ready", "two")] });

    expect(screen.getByText("2 connected instances")).toBeInTheDocument();
  });

  it("configures the instance that was clicked", async () => {
    const user = userEvent.setup();
    const onConfigure = vi.fn();
    const connected = instance("ready");

    renderRow({ instances: [connected] }, { onConfigure });
    await user.click(screen.getByRole("button", { name: "Configure" }));

    expect(onConfigure).toHaveBeenCalledWith(connected);
  });

  it("disables connecting when the provider is no longer available", () => {
    renderRow({ integrationDef: null });

    expect(screen.getByRole("button", { name: "Unavailable" })).toBeDisabled();
  });
});
