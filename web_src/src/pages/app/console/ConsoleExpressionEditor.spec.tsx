import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ConsoleExpressionEditor } from "./ConsoleExpressionEditor";

describe("ConsoleExpressionEditor", () => {
  it("provides the widget CEL adapter without global registration", () => {
    render(
      <ConsoleExpressionEditor
        aria-label="Widget field"
        syntaxProfile="pathOrRaw"
        exampleObj={{ status: "passed" }}
        value="status"
        onChange={vi.fn()}
        showValuePreview
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Preview" }));

    expect(screen.getByText("passed")).toBeInTheDocument();
    expect(screen.queryByText(/adapter is not registered/i)).not.toBeInTheDocument();
  });
});
