import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { NumberFieldRenderer } from "./NumberFieldRenderer";

function createNumberField(defaultValue?: string): ConfigurationField {
  return {
    name: "count",
    label: "Count",
    type: "number",
    defaultValue,
  };
}

describe("NumberFieldRenderer", () => {
  it("does not apply default values when readOnly", () => {
    const handleChange = vi.fn();

    render(<NumberFieldRenderer field={createNumberField("5")} value={undefined} onChange={handleChange} readOnly />);

    expect(handleChange).not.toHaveBeenCalled();
  });
});
