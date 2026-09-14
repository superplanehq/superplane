import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

const showErrorToastMock = vi.fn();
vi.mock("@/lib/toast", () => ({
  showErrorToast: (...args: unknown[]) => showErrorToastMock(...args),
}));

import { CopyableKeyButton } from "./CopyableKeyButton";

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

function renderKey(value = "RF-105") {
  return render(<CopyableKeyButton value={value} />);
}

describe("CopyableKeyButton", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    showErrorToastMock.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("copies the ID and shows a Copied confirmation that reverts", async () => {
    const writeText = mockClipboard(() => Promise.resolve());

    renderKey();

    fireEvent.click(screen.getByTestId("popup-work-order-display-key"));
    await flushPromises();

    expect(writeText).toHaveBeenCalledWith("RF-105");
    expect(screen.getAllByText("Copied").length).toBeGreaterThan(0);

    act(() => {
      vi.advanceTimersByTime(1600);
    });

    expect(screen.queryByText("Copied")).not.toBeInTheDocument();
  });

  it("shows an error toast when the clipboard write fails", async () => {
    mockClipboard(() => Promise.reject(new Error("denied")));

    renderKey();

    fireEvent.click(screen.getByTestId("popup-work-order-display-key"));
    await flushPromises();

    expect(showErrorToastMock).toHaveBeenCalledWith("Failed to copy the task ID.");
    expect(screen.queryByText("Copied")).not.toBeInTheDocument();
  });
});
