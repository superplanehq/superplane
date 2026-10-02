import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";

import { NumberFieldRenderer } from "./NumberFieldRenderer";

function numberField(overrides: Partial<ConfigurationField> = {}): ConfigurationField {
  return {
    name: "minutes",
    type: "number",
    label: "Minutes between triggers",
    ...overrides,
  } as ConfigurationField;
}

/** Renders the field the way forms do: the parent owns the value. */
function ControlledHarness({
  field,
  initialValue,
  onChangeSpy,
}: {
  field: ConfigurationField;
  initialValue?: unknown;
  onChangeSpy: (value: unknown) => void;
}) {
  const [value, setValue] = useState<unknown>(initialValue);
  return (
    <NumberFieldRenderer
      field={field}
      value={value}
      onChange={(next) => {
        onChangeSpy(next);
        setValue(next);
      }}
    />
  );
}

describe("NumberFieldRenderer", () => {
  it("initializes an empty value from the field default on mount", () => {
    const onChange = vi.fn();
    render(<ControlledHarness field={numberField({ defaultValue: "1" })} onChangeSpy={onChange} />);

    expect(onChange).toHaveBeenCalledWith(1);
    expect(screen.getByRole("spinbutton")).toHaveValue(1);
  });

  it("lets the user clear the field without snapping back to the default", () => {
    const onChange = vi.fn();
    render(<ControlledHarness field={numberField({ defaultValue: "1" })} onChangeSpy={onChange} />);

    const input = screen.getByRole("spinbutton");
    fireEvent.change(input, { target: { value: "" } });

    // The cleared state must survive: no effect may re-apply the default mid-edit.
    expect(input).toHaveValue(null);
    expect(onChange).toHaveBeenLastCalledWith(undefined);
  });

  it("lets the user clear and type a new number (the spinner must not be the only way)", () => {
    const onChange = vi.fn();
    render(<ControlledHarness field={numberField({ defaultValue: "1" })} onChangeSpy={onChange} />);

    const input = screen.getByRole("spinbutton");
    fireEvent.change(input, { target: { value: "" } });
    fireEvent.change(input, { target: { value: "5" } });

    expect(input).toHaveValue(5);
    expect(onChange).toHaveBeenLastCalledWith(5);
  });

  it("syncs an external value change into the input", () => {
    const onChange = vi.fn();
    const { rerender } = render(<NumberFieldRenderer field={numberField()} value={7} onChange={onChange} />);
    expect(screen.getByRole("spinbutton")).toHaveValue(7);

    rerender(<NumberFieldRenderer field={numberField()} value={42} onChange={onChange} />);
    expect(screen.getByRole("spinbutton")).toHaveValue(42);
  });
});
