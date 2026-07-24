import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { useCanvas } from "@/hooks/useCanvasData";

import { RunParametersFieldRenderer } from "./RunParametersFieldRenderer";

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
    type: "run-parameters",
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
      <RunParametersFieldRenderer
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
            id: "on-run",
            name: "On Run",
            type: "TYPE_TRIGGER",
            component: "onRun",
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

describe("RunParametersFieldRenderer", () => {
  it("prompts for app and node before loading parameters", () => {
    renderWithTheme(<ControlledRenderer allValues={{ app: "canvas_target" }} />);

    expect(screen.getByText("Choose the target app and node before configuring run parameters.")).toBeTruthy();
  });

  it("renders typed fields when the target node defines parameters", async () => {
    renderWithTheme(
      <ControlledRenderer initialValue={{ message: "hello" }} allValues={{ app: "canvas_target", node: "on-run" }} />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("string-field-message")).toBeTruthy();
    });

    expect(screen.getByText("Message")).toBeTruthy();
    expect(screen.getByDisplayValue("hello")).toBeTruthy();
  });

  it("updates the parameters object when a typed field changes", async () => {
    const user = userEvent.setup();

    renderWithTheme(<ControlledRenderer initialValue={{}} allValues={{ app: "canvas_target", node: "on-run" }} />);

    const input = await screen.findByTestId("string-field-message");
    await user.type(input, "updated");

    await waitFor(() => {
      expect(screen.getByTestId("current-value").textContent).toContain('"message":"updated"');
    });
  });

  it("shows an informational message when the target node has no parameters", async () => {
    mockUseCanvas.mockReturnValue({
      data: {
        id: "canvas_target",
        spec: {
          nodes: [
            {
              id: "on-run-empty",
              name: "Run",
              type: "TYPE_TRIGGER",
              component: "onRun",
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
      <ControlledRenderer initialValue={{ custom: true }} allValues={{ app: "canvas_target", node: "on-run-empty" }} />,
    );

    await waitFor(() => {
      const message = screen.getByTestId("run-parameters-field-parameters");
      expect(message).toHaveTextContent("The trigger you selected does not define any parameters.");
      expect(message).toHaveTextContent(
        "If parameters are needed in your flow, define them in the trigger configuration first.",
      );
      expect(message).toHaveTextContent(
        "Without parameters, the run will still be triggered, but no additional values will be passed.",
      );
      expect(screen.queryByTestId("string-field-message")).toBeNull();
    });
  });
});
