import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";

import { TimezoneFieldRenderer } from "./TimezoneFieldRenderer";

const field: ConfigurationField = {
  name: "timezone",
  label: "Timezone",
  type: "timezone",
};

describe("TimezoneFieldRenderer", () => {
  it("resolves the 'current' placeholder to the browser timezone identifier", () => {
    const onChange = vi.fn();

    render(<TimezoneFieldRenderer field={field} value="current" onChange={onChange} />);

    const resolved = Intl.DateTimeFormat().resolvedOptions().timeZone;
    expect(onChange).toHaveBeenCalledWith(resolved);
    expect(resolved).toMatch(/^[A-Za-z]+(\/[A-Za-z_+-]+)*$/);
  });

  it("resolves an unset value to the browser timezone identifier", () => {
    const onChange = vi.fn();

    render(<TimezoneFieldRenderer field={field} value={undefined} onChange={onChange} />);

    expect(onChange).toHaveBeenCalledWith(Intl.DateTimeFormat().resolvedOptions().timeZone);
  });

  it("offers IANA identifiers so daylight saving is applied", () => {
    render(<TimezoneFieldRenderer field={field} value="America/New_York" onChange={vi.fn()} />);

    const selected = screen.getByText(/America\/New_York/);
    expect(selected).toBeInTheDocument();

    fireEvent.click(selected);
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "Europe/Paris" } });

    expect(screen.getByText(/Europe\/Paris/)).toBeInTheDocument();
  });

  it("keeps a legacy numeric offset selectable", () => {
    const onChange = vi.fn();

    render(<TimezoneFieldRenderer field={field} value="-5" onChange={onChange} />);

    // The stored value stays put rather than being silently rewritten.
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.getByText(/GMT-5 \(fixed offset\)/)).toBeInTheDocument();
  });
});
