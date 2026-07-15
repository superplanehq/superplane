import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { RunSourceControls } from "./MarkdownVariableSourceControls";

describe("RunSourceControls", () => {
  it("drops status selections that cannot match the selected run bucket", () => {
    const onChange = vi.fn();
    render(<RunSourceControls source={{ kind: "run", select: "latest_passed" }} onChange={onChange} />);

    fireEvent.click(screen.getByRole("button", { name: "Toggle run filters" }));
    fireEvent.click(screen.getByText("Failed").closest("label")!);

    expect(onChange).toHaveBeenCalledWith({ kind: "run", select: "latest_passed" });
  });
});
