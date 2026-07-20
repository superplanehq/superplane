import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";
import { useCanvases } from "@/hooks/useCanvasData";

import { AppFieldRenderer } from "./AppFieldRenderer";

vi.mock("react-router-dom", () => ({
  useParams: () => ({ appId: "canvas_current" }),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvases: vi.fn(),
}));

const mockUseCanvases = vi.mocked(useCanvases);

function appField(overrides: Partial<ConfigurationField> = {}): ConfigurationField {
  return {
    name: "app",
    label: "Source app",
    type: "app",
    ...overrides,
  };
}

function ControlledRenderer({
  initialValue,
  field = appField(),
}: {
  initialValue?: string;
  field?: ConfigurationField;
}) {
  const [value, setValue] = useState<string | undefined>(initialValue);
  return (
    <>
      <span data-testid="current-value">{value ?? ""}</span>
      <AppFieldRenderer field={field} value={value} onChange={setValue} organizationId="org-1" />
    </>
  );
}

beforeEach(() => {
  mockUseCanvases.mockReturnValue({
    data: [
      { id: "canvas_current", name: "Current App" },
      { id: "canvas_billing", name: "Billing Alerts" },
      { id: "canvas_onboarding", name: "Customer Onboarding" },
    ],
    isLoading: false,
    error: null,
  } as ReturnType<typeof useCanvases>);
});

describe("AppFieldRenderer", () => {
  it("excludes the current app from selectable options by default", async () => {
    render(<ControlledRenderer />);

    await userEvent.click(screen.getByRole("combobox"));
    expect(screen.getByText("Billing Alerts")).toBeInTheDocument();
    expect(screen.getByText("Customer Onboarding")).toBeInTheDocument();
    expect(screen.queryByText("Current App")).not.toBeInTheDocument();
  });

  it("includes the current app when allowSelf is enabled", async () => {
    render(
      <ControlledRenderer
        field={appField({
          typeOptions: {
            app: {
              allowSelf: true,
            },
          },
        })}
      />,
    );

    await userEvent.click(screen.getByRole("combobox"));
    expect(screen.getByText("Current App")).toBeInTheDocument();
  });

  it("stores the selected app id", async () => {
    render(<ControlledRenderer />);

    await userEvent.click(screen.getByRole("combobox"));
    await userEvent.click(screen.getByText("Billing Alerts"));

    expect(screen.getByTestId("current-value")).toHaveTextContent("canvas_billing");
  });
});
