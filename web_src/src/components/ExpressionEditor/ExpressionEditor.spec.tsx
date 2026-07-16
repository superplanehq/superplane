import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ExpressionAdapter } from "@/lib/expression";
import { ExpressionEditor } from "./ExpressionEditor";
import { getExpressionDialectAdapter, registerExpressionDialect } from "./expressionDialectRegistry";

describe("ExpressionEditor", () => {
  it("uses the expr-lang defaults for wrapped fields", async () => {
    render(
      <ExpressionEditor
        aria-label="Run title"
        exampleObj={{ __root: { data: { name: "DCO" } } }}
        value=""
        onChange={vi.fn()}
        showValuePreview
      />,
    );

    const input = screen.getByRole("textbox", { name: "Run title" });
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "{{ root().data.", selectionStart: "{{ root().data.".length } });

    expect(await screen.findByText("name")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Preview" })).toBeInTheDocument();
  });

  it("routes evaluation and previews through a custom adapter", () => {
    const adapter: ExpressionAdapter = {
      id: "cel",
      evaluate: vi.fn().mockReturnValue({ ok: true, value: "42", formattedValue: "42" }),
      resolveSuggestionValue: vi.fn().mockReturnValue("42"),
      formatResult: vi.fn().mockImplementation((v) => String(v)),
    };

    render(
      <ExpressionEditor
        aria-label="Widget title"
        exampleObj={{ status: "ok" }}
        value="{{ status }}"
        onChange={vi.fn()}
        expressionAdapter={adapter}
        showValuePreview
      />,
    );

    // Toggle preview mode so the adapter is exercised for the rendered value.
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));

    expect(adapter.evaluate).toHaveBeenCalled();
    const evaluateArgs = (adapter.evaluate as ReturnType<typeof vi.fn>).mock.calls.at(0);
    expect(evaluateArgs?.[0]).toContain("status");
  });

  it("passes raw syntax profile through to the underlying editor", () => {
    render(
      <ExpressionEditor
        aria-label="Raw editor"
        exampleObj={{ hits: 3 }}
        value="hits + 1"
        onChange={vi.fn()}
        syntaxProfile="raw"
        showValuePreview
      />,
    );

    // In raw mode the input value is treated as a bare expression, so the
    // preview toggle is present with a quickTip about paths, not `{{`.
    expect(screen.getByRole("button", { name: "Preview" })).toBeInTheDocument();
    expect(screen.getByText(/browse node payloads/i)).toBeInTheDocument();
  });

  it("registerExpressionDialect wires a new default adapter", () => {
    const previousAdapter = getExpressionDialectAdapter("cel");
    const adapter: ExpressionAdapter = {
      id: "cel",
      evaluate: vi.fn().mockReturnValue({ ok: true, value: null, formattedValue: "null" }),
      resolveSuggestionValue: vi.fn(),
      formatResult: vi.fn(),
    };
    const restore = registerExpressionDialect("cel", adapter);

    expect(getExpressionDialectAdapter("cel")).toBe(adapter);

    restore();
    expect(getExpressionDialectAdapter("cel")).toBe(previousAdapter);
  });
});
