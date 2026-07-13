import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { SelectFieldRenderer } from "./SelectFieldRenderer";
import type { ConfigurationField } from "../../api-client";

const baseField: ConfigurationField = {
  name: "encoding",
  label: "Encoding",
  type: "select",
  typeOptions: {
    select: {
      options: [
        { label: "Text", value: "text" },
        { label: "Base64", value: "base64" },
      ],
    },
  },
};

const expressionField: ConfigurationField = {
  ...baseField,
  typeOptions: {
    select: {
      ...baseField.typeOptions!.select!,
      allowExpressions: true,
    },
  },
};

describe("SelectFieldRenderer expressions", () => {
  it("renders no expression toggle when the field does not opt in", () => {
    render(<SelectFieldRenderer field={baseField} value="text" onChange={vi.fn()} allowExpressions />);
    expect(screen.queryByTestId("field-encoding-use-expression")).toBeNull();
  });

  it("renders no expression toggle when expressions are globally disabled", () => {
    render(<SelectFieldRenderer field={expressionField} value="text" onChange={vi.fn()} allowExpressions={false} />);
    expect(screen.queryByTestId("field-encoding-use-expression")).toBeNull();
  });

  it("switches to an expression input via the toggle", () => {
    render(<SelectFieldRenderer field={expressionField} value="text" onChange={vi.fn()} allowExpressions />);
    fireEvent.click(screen.getByTestId("field-encoding-use-expression"));
    expect(screen.getByTestId("field-encoding-expression")).toBeTruthy();
  });

  it("starts in expression mode when the value is an expression", () => {
    render(
      <SelectFieldRenderer
        field={expressionField}
        value="{{ $['Text Prompt'].data.artifacts[0].encoding }}"
        onChange={vi.fn()}
        allowExpressions
      />,
    );
    expect(screen.getByTestId("field-encoding-expression")).toBeTruthy();
  });

  it("returns to the dropdown and clears the value", () => {
    const onChange = vi.fn();
    render(<SelectFieldRenderer field={expressionField} value="{{ expr }}" onChange={onChange} allowExpressions />);
    fireEvent.click(screen.getByTestId("field-encoding-use-options"));
    expect(onChange).toHaveBeenCalledWith(undefined);
  });
});
