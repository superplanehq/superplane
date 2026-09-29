import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { CanvasSpecYamlModal } from "./CanvasSpecYamlModal";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value }: { value?: string }) => <pre data-testid="monaco-editor">{value}</pre>,
}));

vi.mock("@/contexts/useTheme", () => ({
  useTheme: () => ({ preference: "light", resolvedTheme: "light", setPreference: () => undefined }),
}));

describe("CanvasSpecYamlModal", () => {
  it("shows canvas.yaml by default and can switch to console.yaml", async () => {
    const user = userEvent.setup();
    render(
      <CanvasSpecYamlModal
        open
        onOpenChange={vi.fn()}
        canvasYaml={"apiVersion: v1\nkind: Canvas\nmetadata:\n  name: Refund\n"}
        consoleYaml={"apiVersion: v1\nkind: Console\nmetadata:\n  name: Refund\n"}
      />,
    );

    expect(screen.getByTestId("canvas-spec-yaml-modal")).toBeInTheDocument();
    expect(screen.getByText("View YAML")).toBeInTheDocument();
    expect(screen.getByTestId("monaco-editor")).toHaveTextContent("kind: Canvas");

    await user.click(screen.getByTestId("canvas-spec-yaml-tab-console"));

    expect(screen.getByTestId("monaco-editor")).toHaveTextContent("kind: Console");
  });

  it("copies the active YAML", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(<CanvasSpecYamlModal open onOpenChange={vi.fn()} canvasYaml="name: Refund" consoleYaml="kind: Console" />);

    fireEvent.click(screen.getByTestId("canvas-spec-yaml-copy"));

    expect(writeText).toHaveBeenCalledWith("name: Refund");
  });
});
