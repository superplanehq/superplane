import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";

import { NumberFieldRenderer } from "./NumberFieldRenderer";

function numberField(overrides: Partial<ConfigurationField> = {}): ConfigurationField {
  return {
    name: "timeout",
    type: "number",
    label: "Timeout",
    ...overrides,
  } as ConfigurationField;
}

function ControlledNumber({
  field,
  initialValue,
  onChange,
}: {
  field: ConfigurationField;
  initialValue?: unknown;
  onChange?: (value: unknown) => void;
}) {
  const [value, setValue] = useState<unknown>(initialValue);
  return (
    <NumberFieldRenderer
      field={field}
      value={value}
      onChange={(next) => {
        setValue(next);
        onChange?.(next);
      }}
    />
  );
}

describe("NumberFieldRenderer", () => {
  it("applies the default value on first render when no value is present", () => {
    const onChange = vi.fn();
    render(<ControlledNumber field={numberField({ defaultValue: "3" })} onChange={onChange} />);

    expect(onChange).toHaveBeenCalledWith(3);
    expect(screen.getByRole("spinbutton")).toHaveValue(3);
  });

  it("renders the current value", () => {
    render(<ControlledNumber field={numberField()} initialValue={42} />);

    expect(screen.getByRole("spinbutton")).toHaveValue(42);
  });

  it("emits undefined when the field is cleared", () => {
    const onChange = vi.fn();
    render(<ControlledNumber field={numberField({ defaultValue: "3" })} initialValue={3} onChange={onChange} />);

    fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "" } });

    expect(onChange).toHaveBeenLastCalledWith(undefined);
  });

  it("stays empty after being cleared instead of snapping back to the default value", () => {
    render(<ControlledNumber field={numberField({ defaultValue: "3" })} initialValue={3} />);

    fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "" } });

    expect(screen.getByRole("spinbutton")).toHaveValue(null);
  });

  it("emits a number when a valid value is typed", () => {
    const onChange = vi.fn();
    render(<ControlledNumber field={numberField()} onChange={onChange} />);

    fireEvent.change(screen.getByRole("spinbutton"), { target: { value: "7" } });

    expect(onChange).toHaveBeenLastCalledWith(7);
  });

  it("exposes the configured min and max as input attributes", () => {
    render(<ControlledNumber field={numberField({ typeOptions: { number: { min: 2, max: 4 } } })} />);

    const input = screen.getByRole("spinbutton") as HTMLInputElement;
    expect(input.min).toBe("2");
    expect(input.max).toBe("4");
  });
});
