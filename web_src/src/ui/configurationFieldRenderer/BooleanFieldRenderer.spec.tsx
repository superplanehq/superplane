import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ConfigurationField } from "@/api-client";
import { BooleanFieldRenderer } from "./BooleanFieldRenderer";

function createBooleanField(defaultValue?: string): ConfigurationField {
  return {
    name: "enabled",
    label: "Enabled",
    type: "boolean",
    defaultValue,
  };
}

describe("BooleanFieldRenderer", () => {
  it("renders checked when value is true", () => {
    render(<BooleanFieldRenderer field={createBooleanField()} value={true} onChange={vi.fn()} />);

    expect(screen.getByRole("switch")).toHaveAttribute("aria-checked", "true");
  });

  it("renders unchecked when value is false even if default value is true", () => {
    render(<BooleanFieldRenderer field={createBooleanField("true")} value={false} onChange={vi.fn()} />);

    expect(screen.getByRole("switch")).toHaveAttribute("aria-checked", "false");
  });

  it("uses field default value when value is undefined", () => {
    render(<BooleanFieldRenderer field={createBooleanField("true")} value={undefined} onChange={vi.fn()} />);

    expect(screen.getByRole("switch")).toHaveAttribute("aria-checked", "true");
  });

  it("renders unchecked when both value and default value are absent", () => {
    render(<BooleanFieldRenderer field={createBooleanField()} value={undefined} onChange={vi.fn()} />);

    expect(screen.getByRole("switch")).toHaveAttribute("aria-checked", "false");
  });

  it("calls onChange with checked state when toggled", async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(<BooleanFieldRenderer field={createBooleanField()} value={false} onChange={handleChange} />);

    await user.click(screen.getByRole("switch"));

    expect(handleChange).toHaveBeenCalledTimes(1);
    expect(handleChange).toHaveBeenCalledWith(true);
  });
});
