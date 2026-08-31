import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { toTestId } from "@/lib/testID";
import { StringFieldRenderer } from "./StringFieldRenderer";

function createStringField(overrides: Partial<ConfigurationField> = {}): ConfigurationField {
  return {
    name: "customName",
    label: "Custom name",
    type: "string",
    ...overrides,
  };
}

describe("StringFieldRenderer", () => {
  it("coerces a non-string value in the plain Input branch instead of crashing", () => {
    render(<StringFieldRenderer field={createStringField()} value={123} onChange={vi.fn()} />);

    expect(screen.getByTestId(toTestId("string-field-customName"))).toHaveValue("123");
  });

  it("coerces a non-string value when rendering with AutoCompleteInput (allowExpressions)", () => {
    render(
      <StringFieldRenderer
        field={createStringField()}
        value={123}
        onChange={vi.fn()}
        allowExpressions
        autocompleteExampleObj={null}
      />,
    );

    expect(screen.getByTestId(toTestId("string-field-customName"))).toHaveValue("123");
  });

  it("renders an empty string for null/undefined values instead of the literal 'null'/'undefined'", () => {
    render(<StringFieldRenderer field={createStringField()} value={null} onChange={vi.fn()} />);

    expect(screen.getByTestId(toTestId("string-field-customName"))).toHaveValue("");
  });

  it("still renders a normal string value unchanged", () => {
    render(<StringFieldRenderer field={createStringField()} value="hello" onChange={vi.fn()} />);

    expect(screen.getByTestId(toTestId("string-field-customName"))).toHaveValue("hello");
  });
});
