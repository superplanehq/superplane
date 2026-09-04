import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConfigurationField, OrganizationsIntegration } from "@/api-client";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { usePermissions } from "@/contexts/usePermissions";
import { IntegrationFieldRenderer, type IntegrationRefValue } from "./IntegrationFieldRenderer";

const navigateMock = vi.hoisted(() => vi.fn());

vi.mock("react-router-dom", () => ({
  useNavigate: () => navigateMock,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: vi.fn(),
  useAvailableIntegrations: vi.fn(),
  useCreateIntegration: vi.fn(),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: vi.fn(),
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: ({
    open,
    integrationDefinition,
    onCreated,
  }: {
    open: boolean;
    integrationDefinition?: { name?: string } | null;
    onCreated?: (integrationId: string, instanceName: string) => void;
  }) =>
    open ? (
      <div data-testid="integration-create-dialog">
        <span>{integrationDefinition?.name}</span>
        <button type="button" onClick={() => onCreated?.("int_claude_new", "my-new-claude")}>
          finish-connect
        </button>
      </div>
    ) : null,
}));

vi.mock("@/ui/componentSidebar/integrationIcons", () => ({
  IntegrationIcon: ({ integrationName }: { integrationName?: string }) => (
    <span data-testid={`integration-icon-${integrationName ?? "unknown"}`} />
  ),
}));

function createIntegrations(): OrganizationsIntegration[] {
  return [
    {
      metadata: {
        id: "int_github_default",
        name: "github",
        integrationName: "github",
      },
      status: { state: "ready" },
    },
    {
      metadata: {
        id: "int_semaphore",
        name: "my-semaphore",
        integrationName: "semaphore",
      },
      status: { state: "ready" },
    },
    {
      metadata: {
        id: "int_pending",
        name: "Pending GitHub",
        integrationName: "github",
      },
      status: { state: "pending" },
    },
  ];
}

function mockIntegrationsHook(integrations: OrganizationsIntegration[] = createIntegrations()) {
  vi.mocked(useConnectedIntegrations).mockReturnValue({
    data: integrations,
    isLoading: false,
    error: null,
  } as ReturnType<typeof useConnectedIntegrations>);
}

function mockPermissions(canCreate: boolean) {
  vi.mocked(usePermissions).mockReturnValue({
    canAct: (resource: string, action: string) => canCreate && resource === "integrations" && action === "create",
  } as unknown as ReturnType<typeof usePermissions>);
}

function mockIntegrationCatalogHooks() {
  vi.mocked(useAvailableIntegrations).mockReturnValue({
    data: [{ name: "claude", label: "Claude" }],
  } as ReturnType<typeof useAvailableIntegrations>);
  vi.mocked(useCreateIntegration).mockReturnValue({
    mutateAsync: vi.fn(),
    reset: vi.fn(),
  } as unknown as ReturnType<typeof useCreateIntegration>);
}

function createField(): ConfigurationField {
  return {
    name: "integration",
    type: "integration",
    label: "Integration",
    placeholder: "Select integration",
  };
}

function ControlledIntegrationFieldRenderer({ initialValue }: { initialValue: IntegrationRefValue }) {
  const [value, setValue] = React.useState<IntegrationRefValue>(initialValue);

  return (
    <IntegrationFieldRenderer
      field={createField()}
      isRequired
      value={value}
      onChange={setValue}
      organizationId="org_123"
    />
  );
}

describe("IntegrationFieldRenderer", () => {
  beforeAll(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
    Element.prototype.scrollIntoView ??= () => {};
  });

  beforeEach(() => {
    vi.clearAllMocks();
    mockIntegrationsHook();
    mockPermissions(true);
    mockIntegrationCatalogHooks();
    vi.spyOn(window, "open").mockReturnValue(null);
  });

  afterAll(async () => {
    // Radix Select renders its listbox inside a FocusScope that, on unmount,
    // schedules focus restoration with setTimeout(0). Tests here open the
    // Select and leave it mounted, so that timer is still pending when Testing
    // Library unmounts during cleanup. If it fires after Vitest swaps in the
    // next test file's jsdom realm, the callback builds a CustomEvent from the
    // new realm and dispatches it on the old container, throwing
    // "parameter 1 is not of type 'Event'" and failing the whole shard. Flush
    // the pending timer here so it runs inside this file's realm.
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
  });

  it("renders ready integrations and stores installation name on selection", async () => {
    const user = userEvent.setup();
    let latestValue: IntegrationRefValue;

    render(
      <IntegrationFieldRenderer
        field={createField()}
        isRequired
        value={undefined}
        onChange={(value) => {
          latestValue = value;
        }}
        organizationId="org_123"
      />,
    );

    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByText("my-semaphore"));

    expect(latestValue!).toEqual({
      name: "my-semaphore",
    });
  });

  it("filters integrations by field typeOptions.integration", async () => {
    const user = userEvent.setup();

    render(
      <IntegrationFieldRenderer
        field={{
          ...createField(),
          typeOptions: {
            integration: {
              integration: "semaphore",
            },
          },
        }}
        isRequired
        value={undefined}
        onChange={() => {}}
        organizationId="org_123"
      />,
    );

    await user.click(screen.getByRole("combobox"));
    expect(await screen.findByText("my-semaphore")).toBeInTheDocument();
    expect(screen.queryByText("github")).not.toBeInTheDocument();
  });

  it("shows the installation name as-is with the integration icon", async () => {
    const user = userEvent.setup();

    render(
      <ControlledIntegrationFieldRenderer
        initialValue={{
          name: "github",
        }}
      />,
    );

    expect(screen.getByRole("combobox")).toHaveTextContent("github");
    expect(screen.getByTestId("integration-icon-github")).toBeInTheDocument();

    await user.click(screen.getByRole("combobox"));
    expect(await screen.findByText("my-semaphore")).toBeInTheDocument();
    expect(screen.getByTestId("integration-icon-semaphore")).toBeInTheDocument();
  });

  it("disables the picker when there are no integrations and connect is not allowed", () => {
    mockIntegrationsHook([]);
    mockPermissions(false);

    render(
      <IntegrationFieldRenderer
        field={createField()}
        isRequired
        value={undefined}
        onChange={() => {}}
        organizationId="org_123"
      />,
    );

    expect(screen.getByRole("combobox")).toBeDisabled();
    expect(screen.getByText("No integrations available")).toBeInTheDocument();
    expect(screen.getByText(/Connect an integration in Organization settings first/i)).toBeInTheDocument();
  });

  it("opens integration settings in a new tab when an unfiltered field has no integrations", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    mockIntegrationsHook([]);

    render(
      <IntegrationFieldRenderer
        field={createField()}
        isRequired
        value={undefined}
        onChange={onChange}
        organizationId="org_123"
      />,
    );

    const trigger = screen.getByRole("combobox");
    expect(trigger).not.toBeDisabled();
    await user.click(trigger);
    await user.click(await screen.findByRole("option", { name: /connect an integration/i }));

    expect(window.open).toHaveBeenCalledWith("/org_123/settings/integrations", "_blank", "noopener,noreferrer");
    expect(navigateMock).not.toHaveBeenCalled();
    expect(onChange).not.toHaveBeenCalled();
  });

  it("opens the create dialog inline for a type-filtered field and selects the created integration", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    mockIntegrationsHook([]);

    render(
      <IntegrationFieldRenderer
        field={{
          ...createField(),
          typeOptions: {
            integration: {
              integration: "claude",
            },
          },
        }}
        isRequired
        value={undefined}
        onChange={onChange}
        organizationId="org_123"
      />,
    );

    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: /connect an integration/i }));

    expect(await screen.findByTestId("integration-create-dialog")).toBeInTheDocument();
    expect(window.open).not.toHaveBeenCalled();
    expect(navigateMock).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "finish-connect" }));
    expect(onChange).toHaveBeenCalledWith({ name: "my-new-claude" });
  });

  it("names the integration type when a filter leaves no integrations and connect is not allowed", () => {
    mockPermissions(false);

    render(
      <IntegrationFieldRenderer
        field={{
          ...createField(),
          typeOptions: {
            integration: {
              integration: "claude",
            },
          },
        }}
        isRequired
        value={undefined}
        onChange={() => {}}
        organizationId="org_123"
      />,
    );

    expect(screen.getByRole("combobox")).toBeDisabled();
    expect(screen.getByText("No Claude integrations available")).toBeInTheDocument();
  });

  it("keeps listing ready integrations and offers connect after a separator when integrations exist", async () => {
    const user = userEvent.setup();

    render(
      <IntegrationFieldRenderer
        field={createField()}
        isRequired
        value={undefined}
        onChange={() => {}}
        organizationId="org_123"
      />,
    );

    await user.click(screen.getByRole("combobox"));

    expect(await screen.findByText("github")).toBeInTheDocument();
    expect(screen.getByText("my-semaphore")).toBeInTheDocument();
    expect(screen.getByTestId("integration-picker-connect-option")).toBeInTheDocument();
  });
});
