import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { toTestId } from "@/lib/testID";
import { ExpressionFieldRenderer } from "./ExpressionFieldRenderer";

function createExpressionField(overrides: Partial<ConfigurationField> = {}): ConfigurationField {
  return {
    name: "condition",
    label: "Condition",
    type: "expression",
    ...overrides,
  };
}

describe("ExpressionFieldRenderer", () => {
  it("coerces a non-string value in the plain Input branch instead of crashing", () => {
    render(<ExpressionFieldRenderer field={createExpressionField()} value={42} onChange={vi.fn()} />);

    expect(screen.getByTestId(toTestId("expression-field-condition"))).toHaveValue("42");
  });

  it("coerces a non-string value when rendering with AutoCompleteInput (allowExpressions)", () => {
    render(
      <ExpressionFieldRenderer
        field={createExpressionField()}
        value={42}
        onChange={vi.fn()}
        allowExpressions
        autocompleteExampleObj={null}
      />,
    );

    expect(screen.getByTestId(toTestId("expression-field-condition"))).toHaveValue("42");
  });

  it("falls back to a coerced default value when value is undefined", () => {
    render(
      <ExpressionFieldRenderer
        field={createExpressionField({ defaultValue: true })}
        value={undefined}
        onChange={vi.fn()}
      />,
    );

    expect(screen.getByTestId(toTestId("expression-field-condition"))).toHaveValue("true");
  });

  it("still renders a normal string value unchanged", () => {
    render(<ExpressionFieldRenderer field={createExpressionField()} value="1 == 1" onChange={vi.fn()} />);

    expect(screen.getByTestId(toTestId("expression-field-condition"))).toHaveValue("1 == 1");
  });
});
