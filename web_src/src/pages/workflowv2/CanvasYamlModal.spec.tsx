import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CanvasYamlModal } from "./CanvasYamlModal";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

describe("CanvasYamlModal", () => {
  const defaultProps = {
    open: true,
    onOpenChange: vi.fn(),
    yamlText: "name: test-canvas",
    filename: "test-canvas.yaml",
  };

  it("uses the provided copy handler for copy feedback and error handling", () => {
    const onCopy = vi.fn();

    render(<CanvasYamlModal {...defaultProps} onCopy={onCopy} />);

    fireEvent.click(screen.getByRole("button", { name: "Copy" }));

    expect(onCopy).toHaveBeenCalledTimes(1);
  });

  it("hides the copy button when no copy handler is provided", () => {
    render(<CanvasYamlModal {...defaultProps} />);

    expect(screen.queryByRole("button", { name: "Copy" })).not.toBeInTheDocument();
  });
});
