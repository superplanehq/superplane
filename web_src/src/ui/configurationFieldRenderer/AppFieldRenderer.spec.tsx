import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";
import { useCanvas, useCanvases } from "@/hooks/useCanvasData";
import { useFactoryApps } from "@/hooks/useFactoryData";

import { AppFieldRenderer } from "./AppFieldRenderer";

vi.mock("react-router", () => ({
  useParams: () => ({ appId: "canvas_current" }),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvases: vi.fn(),
  useCanvas: vi.fn(),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryApps: vi.fn(),
}));

const mockUseCanvases = vi.mocked(useCanvases);
const mockUseCanvas = vi.mocked(useCanvas);
const mockUseFactoryApps = vi.mocked(useFactoryApps);

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
  mockUseCanvas.mockReturnValue({
    data: undefined,
    isLoading: false,
    error: null,
  } as ReturnType<typeof useCanvas>);

  mockUseFactoryApps.mockReturnValue({
    data: [],
    isLoading: false,
    error: null,
  } as unknown as ReturnType<typeof useFactoryApps>);

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

  it("lists factory apps when configuring on a factory-owned canvas", async () => {
    mockUseCanvas.mockReturnValue({
      data: {
        metadata: {
          factoryId: "factory_refunds",
        },
      },
      isLoading: false,
      error: null,
    } as ReturnType<typeof useCanvas>);

    mockUseFactoryApps.mockReturnValue({
      data: [
        { id: "canvas_current", name: "Current Factory App" },
        { id: "canvas_planner", name: "Refund Planner" },
      ],
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useFactoryApps>);

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
    expect(screen.getByText("Current Factory App")).toBeInTheDocument();
    expect(screen.getByText("Refund Planner")).toBeInTheDocument();
    expect(screen.queryByText("Billing Alerts")).not.toBeInTheDocument();
  });

  it("stores the selected app id", async () => {
    render(<ControlledRenderer />);

    await userEvent.click(screen.getByRole("combobox"));
    await userEvent.click(screen.getByText("Billing Alerts"));

    expect(screen.getByTestId("current-value")).toHaveTextContent("canvas_billing");
  });

  it("waits for the current canvas before showing org apps", () => {
    mockUseCanvas.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as ReturnType<typeof useCanvas>);

    render(<ControlledRenderer />);

    expect(screen.getByText("Loading apps...")).toBeInTheDocument();
    expect(screen.queryByText("Billing Alerts")).not.toBeInTheDocument();
  });

  it("shows an error when the current canvas fails to load", () => {
    mockUseCanvas.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error("Canvas unavailable"),
    } as ReturnType<typeof useCanvas>);

    render(<ControlledRenderer />);

    expect(screen.getByText("Failed to load apps: Canvas unavailable")).toBeInTheDocument();
    expect(screen.queryByText("Billing Alerts")).not.toBeInTheDocument();
  });
});
