import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";
import { useIntegrationResources } from "@/hooks/useIntegrations";

import { IntegrationResourceFieldRenderer } from "./IntegrationResourceFieldRenderer";

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: vi.fn(),
}));

const mockUseIntegrationResources = vi.mocked(useIntegrationResources);

const EXPRESSION_VALUE = '{{ $["node"].value }}';

function resourceField(): ConfigurationField {
  return {
    name: "channel",
    label: "Channel",
    type: "integration-resource",
    typeOptions: {
      resource: {
        type: "channel",
      },
    },
  };
}

function ControlledRenderer({ initialValue }: { initialValue?: string }) {
  const [value, setValue] = useState<string | string[] | undefined>(initialValue);
  return (
    <>
      <span data-testid="current-value">{typeof value === "string" ? value : ""}</span>
      <IntegrationResourceFieldRenderer
        field={resourceField()}
        value={value}
        onChange={setValue}
        organizationId="org-1"
        integrationId="int-1"
        allowExpressions
      />
    </>
  );
}

beforeEach(() => {
  mockUseIntegrationResources.mockReturnValue({
    data: [
      { id: "resource-1", name: "Resource One" },
      { id: "resource-2", name: "Resource Two" },
    ],
    isLoading: false,
    error: null,
  } as unknown as ReturnType<typeof useIntegrationResources>);
});

describe("IntegrationResourceFieldRenderer", () => {
  it("preserves a fixed value when toggling to Expression and back", async () => {
    const user = userEvent.setup();
    render(<ControlledRenderer initialValue="resource-1" />);

    expect(screen.getByTestId("current-value").textContent).toBe("resource-1");

    await user.click(screen.getByRole("tab", { name: "Expression" }));

    expect(screen.getByTestId("current-value").textContent).toBe("resource-1");
    expect(screen.getByRole("textbox")).toHaveValue("resource-1");

    await user.click(screen.getByRole("tab", { name: "Fixed" }));

    expect(screen.getByTestId("current-value").textContent).toBe("resource-1");
  });

  it("preserves an expression value when toggling to Fixed and back", async () => {
    const user = userEvent.setup();
    render(<ControlledRenderer initialValue={EXPRESSION_VALUE} />);

    expect(screen.getByRole("textbox")).toHaveValue(EXPRESSION_VALUE);

    await user.click(screen.getByRole("tab", { name: "Fixed" }));

    expect(screen.getByTestId("current-value").textContent).toBe(EXPRESSION_VALUE);

    await user.click(screen.getByRole("tab", { name: "Expression" }));

    expect(screen.getByRole("textbox")).toHaveValue(EXPRESSION_VALUE);
  });

  it("never clears the value via onChange when switching modes", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();
    render(
      <IntegrationResourceFieldRenderer
        field={resourceField()}
        value="resource-1"
        onChange={handleChange}
        organizationId="org-1"
        integrationId="int-1"
        allowExpressions
      />,
    );

    await user.click(screen.getByRole("tab", { name: "Expression" }));
    await user.click(screen.getByRole("tab", { name: "Fixed" }));

    expect(handleChange).not.toHaveBeenCalledWith(undefined);
  });
});
