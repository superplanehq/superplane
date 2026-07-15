import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { useCanvas } from "@/hooks/useCanvasData";

import { InvocationParametersFieldRenderer } from "./InvocationParametersFieldRenderer";

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: vi.fn(),
}));

const mockUseCanvas = vi.mocked(useCanvas);

function renderWithTheme(ui: ReactNode) {
  return render(<ThemeProvider>{ui}</ThemeProvider>);
}

function parametersField(): ConfigurationField {
  return {
    name: "parameters",
    label: "Parameters",
    type: "invocation-parameters",
    required: true,
  };
}

function ControlledRenderer({
  initialValue,
  allValues,
}: {
  initialValue?: Record<string, unknown>;
  allValues?: Record<string, unknown>;
}) {
  const [value, setValue] = useState<unknown>(initialValue);
  return (
    <>
      <span data-testid="current-value">{JSON.stringify(value ?? null)}</span>
      <InvocationParametersFieldRenderer
        field={parametersField()}
        value={value}
        onChange={setValue}
        allValues={allValues}
        organizationId="org-1"
      />
    </>
  );
}

beforeEach(() => {
  mockUseCanvas.mockReturnValue({
    data: {
      id: "canvas_target",
      spec: {
        nodes: [
          {
            id: "on-invoke",
            name: "On Invoke",
            type: "TYPE_TRIGGER",
            component: "onInvoke",
            configuration: {
              parameters: [
                {
                  type: "string",
                  name: "message",
                  label: "Message",
                  description: "Message body",
                  required: true,
                },
              ],
            },
          },
        ],
      },
    },
    isLoading: false,
    error: null,
  } as unknown as ReturnType<typeof useCanvas>);
});

describe("InvocationParametersFieldRenderer", () => {
  it("prompts for app and node before loading parameters", () => {
    renderWithTheme(<ControlledRenderer allValues={{ app: "canvas_target" }} />);

    expect(screen.getByText("Choose the target app and node before configuring invocation parameters.")).toBeTruthy();
  });

  it("renders typed fields when the target node defines parameters", async () => {
    renderWithTheme(
      <ControlledRenderer
        initialValue={{ message: "hello" }}
        allValues={{ app: "canvas_target", node: "on-invoke" }}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("string-field-message")).toBeTruthy();
    });

    expect(screen.getByText("Message")).toBeTruthy();
    expect(screen.getByDisplayValue("hello")).toBeTruthy();
  });

  it("updates the parameters object when a typed field changes", async () => {
    const user = userEvent.setup();

    renderWithTheme(<ControlledRenderer initialValue={{}} allValues={{ app: "canvas_target", node: "on-invoke" }} />);

    const input = await screen.findByTestId("string-field-message");
    await user.type(input, "updated");

    await waitFor(() => {
      expect(screen.getByTestId("current-value").textContent).toContain('"message":"updated"');
    });
  });

  it("falls back to JSON editing when the target node has no parameters", async () => {
    mockUseCanvas.mockReturnValue({
      data: {
        id: "canvas_target",
        spec: {
          nodes: [
            {
              id: "on-invoke",
              name: "On Invoke",
              type: "TYPE_TRIGGER",
              component: "onInvoke",
              configuration: {
                parameters: [],
              },
            },
          ],
        },
      },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useCanvas>);

    renderWithTheme(
      <ControlledRenderer initialValue={{ custom: true }} allValues={{ app: "canvas_target", node: "on-invoke" }} />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("invocation-parameters-field-parameters")).toBeTruthy();
      expect(screen.queryByTestId("string-field-message")).toBeNull();
    });
  });
});
