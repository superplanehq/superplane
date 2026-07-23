import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConfigurationField, OrganizationsIntegration } from "@/api-client";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { IntegrationFieldRenderer, type IntegrationRefValue } from "./IntegrationFieldRenderer";

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: vi.fn(),
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
    vi.mocked(useConnectedIntegrations).mockReturnValue({
      data: createIntegrations(),
      isLoading: false,
      error: null,
    } as ReturnType<typeof useConnectedIntegrations>);
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
});
