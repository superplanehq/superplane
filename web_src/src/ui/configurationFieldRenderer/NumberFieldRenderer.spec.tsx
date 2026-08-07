import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { NumberFieldRenderer } from "./NumberFieldRenderer";

function createNumberField(defaultValue?: string): ConfigurationField {
  return {
    name: "minutesBetweenTriggers",
    label: "Minutes between triggers",
    type: "number",
    defaultValue,
  };
}

describe("NumberFieldRenderer", () => {
  it("initializes onChange with the default value once, on mount, when value is undefined", () => {
    const handleChange = vi.fn();

    const { rerender } = render(
      <NumberFieldRenderer field={createNumberField("1")} value={undefined} onChange={handleChange} />,
    );

    // The component signals the resolved default to its (controlling) parent...
    expect(handleChange).toHaveBeenCalledWith(1);

    // ...and once the parent feeds that value back down as a prop, it's displayed.
    rerender(<NumberFieldRenderer field={createNumberField("1")} value={1} onChange={handleChange} />);
    expect(screen.getByRole("spinbutton")).toHaveValue(1);
  });

  it("renders empty when both value and default value are absent", () => {
    render(<NumberFieldRenderer field={createNumberField()} value={undefined} onChange={vi.fn()} />);

    expect(screen.getByRole("spinbutton")).toHaveValue(null);
  });

  it("allows clearing the field and typing a new value without snapping back to the default", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    const { rerender } = render(
      <NumberFieldRenderer field={createNumberField("1")} value={1} onChange={handleChange} />,
    );

    const input = screen.getByRole("spinbutton");
    await user.clear(input);

    // Clearing must emit undefined, not silently re-apply the default.
    expect(handleChange).toHaveBeenLastCalledWith(undefined);

    // Once the parent re-renders with the now-undefined value, the field
    // must stay empty (not snap back to the default) so the user can type.
    rerender(<NumberFieldRenderer field={createNumberField("1")} value={undefined} onChange={handleChange} />);
    expect(screen.getByRole("spinbutton")).toHaveValue(null);

    await user.type(input, "5");

    expect(handleChange).toHaveBeenLastCalledWith(5);
  });

  it("does not re-apply the default value on every render once initialized", () => {
    const handleChange = vi.fn();

    const { rerender } = render(
      <NumberFieldRenderer field={createNumberField("1")} value={undefined} onChange={handleChange} />,
    );

    // Initial mount is allowed to apply the default once.
    expect(handleChange).toHaveBeenCalledTimes(1);
    handleChange.mockClear();

    // A later render with value still undefined (e.g. after the user cleared
    // the field) must NOT re-trigger the default-value initialization.
    rerender(<NumberFieldRenderer field={createNumberField("1")} value={undefined} onChange={handleChange} />);
    expect(handleChange).not.toHaveBeenCalled();
  });
});
