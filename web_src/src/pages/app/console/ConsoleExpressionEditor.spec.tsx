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

  it("prevents newlines in console fields by default", () => {
    render(<ConsoleExpressionEditor aria-label="Widget field" exampleObj={{}} value="status" onChange={vi.fn()} />);

    const event = new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true });
    screen.getByRole("textbox", { name: "Widget field" }).dispatchEvent(event);

    expect(event.defaultPrevented).toBe(true);
  });

  it("allows newlines when explicitly enabled", () => {
    render(
      <ConsoleExpressionEditor
        aria-label="Widget body"
        exampleObj={{}}
        value="First line"
        onChange={vi.fn()}
        allowNewlines
      />,
    );

    const event = new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true });
    screen.getByRole("textbox", { name: "Widget body" }).dispatchEvent(event);

    expect(event.defaultPrevented).toBe(false);
  });
});
