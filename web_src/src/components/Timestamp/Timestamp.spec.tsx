import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Timestamp } from "./Timestamp";

describe("Timestamp", () => {
  const iso = "2026-06-02T10:01:10.561Z";

  function getTimeByDateTime(dateTime: string): HTMLElement {
    const time = document.querySelector(`time[datetime="${dateTime}"]`);
    expect(time).toBeInstanceOf(HTMLElement);
    return time as HTMLElement;
  }

  it("renders an absolute label with a machine-readable dateTime", () => {
    render(<Timestamp date={iso} />);
    const time = getTimeByDateTime(iso);
    expect(time.tagName).toBe("TIME");
    expect(time).toHaveAttribute("dateTime", iso);
  });

  it("renders the dashed underline hint by default and omits it when disabled", () => {
    const { rerender } = render(<Timestamp date={iso} />);
    expect(getTimeByDateTime(iso).closest("span")).toHaveClass("decoration-dashed");

    rerender(<Timestamp date={iso} withHint={false} />);
    expect(getTimeByDateTime(iso).closest("span")).not.toHaveClass("decoration-dashed");
  });

  it("renders past times as '… ago' when display is relative", () => {
    render(<Timestamp date={new Date(Date.now() - 5000)} display="relative" />);
    expect(screen.getByText(/ago$/)).toBeInTheDocument();
  });

  it("renders future times as 'in …' instead of clamping to zero", () => {
    render(<Timestamp date={new Date(Date.now() + 3 * 60 * 60 * 1000)} display="relative" />);
    expect(screen.getByText(/^in /)).toBeInTheDocument();
  });

  it("renders a compact abbreviated relative label without a suffix", () => {
    render(
      <Timestamp
        date={new Date(Date.now() - 5 * 60 * 1000)}
        display="relative"
        relativeStyle="abbreviated"
        includeAgo={false}
      />,
    );
    const time = screen.getByText("5m");
    expect(time.tagName).toBe("TIME");
  });

  it("renders a compact abbreviated relative label with an 'ago' suffix", () => {
    render(<Timestamp date={new Date(Date.now() - 5 * 60 * 1000)} display="relative" relativeStyle="abbreviated" />);
    expect(screen.getByText("5m ago")).toBeInTheDocument();
  });

  it("renders abbreviated future times as 'in …' instead of clamping to zero", () => {
    render(
      <Timestamp
        date={new Date(Date.now() + 3 * 60 * 60 * 1000)}
        display="relative"
        relativeStyle="abbreviated"
        includeAgo={false}
      />,
    );
    expect(screen.getByText("in 3h")).toBeInTheDocument();
  });

  it("renders the fallback for missing or invalid dates", () => {
    const { rerender } = render(<Timestamp date={null} fallback={<span>—</span>} />);
    expect(screen.getByText("—")).toBeInTheDocument();

    rerender(<Timestamp date="not-a-date" fallback={<span>n/a</span>} />);
    expect(screen.getByText("n/a")).toBeInTheDocument();
  });

  it("renders a date-only label without time-of-day when display is date", () => {
    render(<Timestamp date={iso} display="date" />);
    const time = getTimeByDateTime(iso);
    const text = time.textContent ?? "";
    expect(text).toMatch(/Jun/);
    expect(text).toMatch(/2026/);
    // Date-only mode intentionally drops the HH:MM segment.
    expect(text).not.toMatch(/\d{2}:\d{2}/);
  });

  it("accepts datetime as an alias for absolute", () => {
    render(<Timestamp date={iso} display="datetime" />);
    const time = getTimeByDateTime(iso);
    expect(time.textContent ?? "").toMatch(/\d{2}:\d{2}/);
  });
});
