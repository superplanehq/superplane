import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const showErrorToastMock = vi.fn();
vi.mock("@/lib/toast", () => ({
  showErrorToast: (...args: unknown[]) => showErrorToastMock(...args),
}));

import { CopyLinkButton } from "./CopyLinkButton";

function mockClipboard(impl: (text: string) => Promise<void>) {
  const writeText = vi.fn(impl);
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText },
  });
  return writeText;
}

async function flushPromises() {
  await act(async () => {
    await Promise.resolve();
  });
}

describe("CopyLinkButton", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    showErrorToastMock.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("copies the given url and shows a Copied confirmation that reverts", async () => {
    const writeText = mockClipboard(() => Promise.resolve());

    render(<CopyLinkButton url="https://example.test/work-order/42" />);

    fireEvent.click(screen.getByTestId("work-order-copy-link-button"));
    await flushPromises();

    expect(writeText).toHaveBeenCalledWith("https://example.test/work-order/42");
    expect(screen.getAllByText("Copied").length).toBeGreaterThan(0);

    act(() => {
      vi.advanceTimersByTime(1600);
    });

    expect(screen.queryByText("Copied")).not.toBeInTheDocument();
  });

  it("falls back to the current page URL when no url is given", async () => {
    const writeText = mockClipboard(() => Promise.resolve());

    render(<CopyLinkButton />);

    fireEvent.click(screen.getByTestId("work-order-copy-link-button"));
    await flushPromises();

    expect(writeText).toHaveBeenCalledWith(window.location.href);
  });

  it("shows an error toast when the clipboard write fails", async () => {
    mockClipboard(() => Promise.reject(new Error("denied")));

    render(<CopyLinkButton url="https://example.test/work-order/42" />);

    fireEvent.click(screen.getByTestId("work-order-copy-link-button"));
    await flushPromises();

    expect(showErrorToastMock).toHaveBeenCalledWith("Failed to copy link.");
    expect(screen.queryByText("Copied")).not.toBeInTheDocument();
  });

  it("uses the given testId and ariaLabel", () => {
    render(<CopyLinkButton testId="popup-work-order-copy-link-button" ariaLabel="Copy link to mission" />);

    expect(screen.getByTestId("popup-work-order-copy-link-button")).toHaveAttribute(
      "aria-label",
      "Copy link to mission",
    );
  });
});
