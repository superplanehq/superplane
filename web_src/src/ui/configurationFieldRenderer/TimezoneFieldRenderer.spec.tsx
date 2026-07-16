import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { resolveTimezoneDisplayValue } from "./timezoneDisplayValue";
import { TimezoneFieldRenderer } from "./TimezoneFieldRenderer";

describe("resolveTimezoneDisplayValue", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("resolves unset values to the user's timezone offset", () => {
    vi.spyOn(Date.prototype, "getTimezoneOffset").mockReturnValue(300);

    expect(resolveTimezoneDisplayValue(undefined)).toBe("-5");
    expect(resolveTimezoneDisplayValue(null)).toBe("-5");
    expect(resolveTimezoneDisplayValue("current")).toBe("-5");
  });

  it("keeps explicit timezone values", () => {
    expect(resolveTimezoneDisplayValue("2")).toBe("2");
  });
});

describe("TimezoneFieldRenderer", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows the user's timezone in read-only mode when unset", () => {
    vi.spyOn(Date.prototype, "getTimezoneOffset").mockReturnValue(300);

    render(
      <TimezoneFieldRenderer
        field={{ name: "timezone", label: "Timezone", type: "timezone" }}
        value={undefined}
        onChange={vi.fn()}
        readOnly
      />,
    );

    expect(screen.getByText("GMT-5 (New York, Toronto)")).toBeInTheDocument();
  });
});
