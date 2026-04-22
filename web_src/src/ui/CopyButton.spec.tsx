import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CopyButton } from "./CopyButton";

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

describe("CopyButton", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("labeled variant toggles to Copied! on click and resets after the timeout", async () => {
    const writeText = mockClipboard(() => Promise.resolve());

    render(
      <CopyButton variant="button" text="secret-token">
        Copy
      </CopyButton>,
    );

    expect(screen.getByRole("button", { name: /^copy$/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button"));
    await flushPromises();

    expect(writeText).toHaveBeenCalledWith("secret-token");
    expect(screen.getByRole("button", { name: /copied!/i })).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(screen.getByRole("button", { name: /^copy$/i })).toBeInTheDocument();
  });

  it("renders a custom copiedLabel in place of the default", async () => {
    mockClipboard(() => Promise.resolve());

    render(
      <CopyButton variant="button" text="secret-token" copiedLabel="Token copied">
        Copy token
      </CopyButton>,
    );

    fireEvent.click(screen.getByRole("button"));
    await flushPromises();

    expect(screen.getByRole("button", { name: /token copied/i })).toBeInTheDocument();
  });

  it("icon variant toggles its aria-label to 'Copied to clipboard' after a click", async () => {
    mockClipboard(() => Promise.resolve());

    render(<CopyButton text="secret-token" />);

    fireEvent.click(screen.getByRole("button", { name: "Copy to clipboard" }));
    await flushPromises();

    expect(screen.getByRole("button", { name: "Copied to clipboard" })).toBeInTheDocument();
  });

  it("invokes onCopyError and leaves the idle label in place when clipboard fails", async () => {
    mockClipboard(() => Promise.reject(new Error("denied")));
    const onCopyError = vi.fn();

    render(
      <CopyButton variant="button" text="secret-token" onCopyError={onCopyError}>
        Copy
      </CopyButton>,
    );

    fireEvent.click(screen.getByRole("button"));
    await flushPromises();

    expect(onCopyError).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: /^copy$/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /copied!/i })).not.toBeInTheDocument();
  });
});
