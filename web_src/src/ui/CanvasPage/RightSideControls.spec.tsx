import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";
import type { ComponentProps } from "react";

import { TooltipProvider } from "@/components/ui/tooltip";

import { RightSideControls } from "./RightSideControls";

function renderControls(props: ComponentProps<typeof RightSideControls>) {
  return render(
    <TooltipProvider>
      <RightSideControls {...props} />
    </TooltipProvider>,
  );
}

describe("RightSideControls", () => {
  it("shows a YAML button next to Add Note in canvas edit mode", async () => {
    const user = userEvent.setup();
    const onOpenSpecYaml = vi.fn();

    renderControls({
      mode: "edit",
      canvasEditControls: true,
      onOpenSpecYaml,
    });

    expect(screen.getByTestId("canvas-add-component-button")).toBeInTheDocument();
    expect(screen.getByTestId("add-note-button")).toBeInTheDocument();

    await user.click(screen.getByTestId("canvas-spec-yaml-button"));

    expect(onOpenSpecYaml).toHaveBeenCalledTimes(1);
  });

  it("hides the YAML button when no open handler is provided", () => {
    renderControls({
      mode: "edit",
      canvasEditControls: true,
    });

    expect(screen.queryByTestId("canvas-spec-yaml-button")).not.toBeInTheDocument();
  });
});
