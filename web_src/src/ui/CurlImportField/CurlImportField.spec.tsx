import { render, screen, fireEvent, act } from "@testing-library/react";
import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";
import { CurlImportField } from "./CurlImportField";

describe("CurlImportField", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("renders empty textarea initially", () => {
    render(<CurlImportField onApply={vi.fn()} />);
    expect(screen.getByLabelText("Import from curl (optional)")).toHaveValue("");
  });

  it("shows success message on valid curl", () => {
    render(<CurlImportField onApply={vi.fn()} />);

    fireEvent.change(screen.getByLabelText("Import from curl (optional)"), {
      target: { value: "curl https://api.example.com" },
    });
    act(() => {
      vi.advanceTimersByTime(350);
    });

    expect(screen.getByText("Parsed successfully. Review imported values below.")).toBeInTheDocument();
  });

  it("shows warning for partially parsed curl", () => {
    render(<CurlImportField onApply={vi.fn()} />);

    fireEvent.change(screen.getByLabelText("Import from curl (optional)"), {
      target: { value: "curl https://api.example.com --cert cert.pem" },
    });
    act(() => {
      vi.advanceTimersByTime(350);
    });

    expect(screen.getByText(/Filled what we could/)).toBeInTheDocument();
  });

  it("shows error for invalid input", () => {
    render(<CurlImportField onApply={vi.fn()} />);

    fireEvent.change(screen.getByLabelText("Import from curl (optional)"), {
      target: { value: "not-a-curl" },
    });
    act(() => {
      vi.advanceTimersByTime(350);
    });

    expect(screen.getByText("Invalid curl command")).toBeInTheDocument();
  });

  it("calls onApply with parsed config on valid input", () => {
    const handleApply = vi.fn();
    render(<CurlImportField onApply={handleApply} />);

    fireEvent.change(screen.getByLabelText("Import from curl (optional)"), {
      target: { value: "curl https://api.example.com/users" },
    });
    act(() => {
      vi.advanceTimersByTime(350);
    });

    expect(handleApply).toHaveBeenCalledWith(
      expect.objectContaining({
        method: "GET",
        url: "https://api.example.com/users",
      }),
    );
  });

  it("debounces parsing on rapid input", () => {
    const handleApply = vi.fn();
    render(<CurlImportField onApply={handleApply} />);

    const input = screen.getByLabelText("Import from curl (optional)");
    fireEvent.change(input, { target: { value: "curl https://api.example.com" } });
    fireEvent.change(input, { target: { value: "curl https://api.example.com/users" } });

    act(() => {
      vi.advanceTimersByTime(100);
    });
    expect(handleApply).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(250);
    });
    expect(handleApply).toHaveBeenCalledTimes(1);
  });

  it("does not call onApply when disabled", () => {
    const handleApply = vi.fn();
    render(<CurlImportField onApply={handleApply} disabled />);

    fireEvent.change(screen.getByLabelText("Import from curl (optional)"), {
      target: { value: "curl https://api.example.com" },
    });
    act(() => {
      vi.advanceTimersByTime(350);
    });

    expect(handleApply).not.toHaveBeenCalled();
  });

  it("does not call onApply for empty input and clears validation state", () => {
    const handleApply = vi.fn();
    render(<CurlImportField onApply={handleApply} />);

    const input = screen.getByLabelText("Import from curl (optional)");
    fireEvent.change(input, { target: { value: "curl https://api.example.com" } });
    act(() => {
      vi.advanceTimersByTime(350);
    });
    expect(screen.getByText("Parsed successfully. Review imported values below.")).toBeInTheDocument();

    fireEvent.change(input, { target: { value: "" } });
    act(() => {
      vi.advanceTimersByTime(350);
    });
    expect(handleApply).toHaveBeenCalledTimes(1);
    expect(screen.queryByText("Parsed successfully. Review imported values below.")).not.toBeInTheDocument();
  });

  it("handles paste event with null clipboard data", () => {
    render(<CurlImportField onApply={vi.fn()} />);
    const input = screen.getByLabelText("Import from curl (optional)");

    expect(() => {
      fireEvent.paste(input, { clipboardData: null });
    }).not.toThrow();
  });

  it("has accessible label and aria-describedby for errors", () => {
    render(<CurlImportField onApply={vi.fn()} />);

    const input = screen.getByLabelText("Import from curl (optional)");
    fireEvent.change(input, { target: { value: "invalid" } });
    act(() => {
      vi.advanceTimersByTime(350);
    });

    expect(input).toHaveAttribute("aria-describedby", "curl-import-feedback");
    expect(screen.getByText("Invalid curl command")).toBeInTheDocument();
  });
});
