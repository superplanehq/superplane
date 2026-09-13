import { StrictMode } from "react";
import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { resetStreamMemoryForTests, StreamingText } from "./StreamingText";

function renderStream(content: string, extra: { contentKey?: string; memoryKey?: string; ready?: boolean } = {}) {
  return (
    <StreamingText content={content} contentKey={extra.contentKey ?? content} memoryKey={extra.memoryKey} ready={extra.ready}>
      {(visible) => <span data-testid="stream-visible">{visible}</span>}
    </StreamingText>
  );
}

describe("StreamingText", () => {
  beforeEach(() => {
    resetStreamMemoryForTests();
    vi.useFakeTimers();
    document.documentElement.style.setProperty("--stream-gap", "90ms");
  });

  afterEach(() => {
    vi.useRealTimers();
    document.documentElement.style.removeProperty("--stream-gap");
    resetStreamMemoryForTests();
  });

  it("leaves the first paint as the full file", () => {
    render(<StrictMode>{renderStream("hello world")}</StrictMode>);

    expect(screen.getByTestId("stream-visible")).toHaveTextContent("hello world");
    expect(screen.getByTestId("stream-visible").parentElement).not.toHaveAttribute("data-streaming");
  });

  it("writes the new file from the top under StrictMode", () => {
    const { rerender } = render(<StrictMode>{renderStream("hello world")}</StrictMode>);

    rerender(<StrictMode>{renderStream("clearer empty state")}</StrictMode>);

    expect(screen.getByTestId("stream-visible")).toHaveTextContent("clearer");
    expect(screen.getByTestId("stream-visible")).not.toHaveTextContent("state");
    expect(screen.getByTestId("stream-visible").parentElement).toHaveAttribute("data-streaming");
  });

  it("keeps writing after a later render with the same copy", () => {
    const { rerender } = render(<StrictMode>{renderStream("one two three")}</StrictMode>);

    rerender(<StrictMode>{renderStream("four five six")}</StrictMode>);
    expect(screen.getByTestId("stream-visible")).toHaveTextContent("four");
    expect(screen.getByTestId("stream-visible")).not.toHaveTextContent("five");

    rerender(<StrictMode>{renderStream("four five six")}</StrictMode>);
    act(() => {
      vi.advanceTimersByTime(90);
    });
    expect(screen.getByTestId("stream-visible")).toHaveTextContent("four five");
    expect(screen.getByTestId("stream-visible")).not.toHaveTextContent("six");
  });

  it("does not stream the first value after the pane becomes ready", () => {
    const { rerender } = render(
      <StrictMode>{renderStream("old summary", { ready: false })}</StrictMode>,
    );

    rerender(<StrictMode>{renderStream("new spec words", { ready: true })}</StrictMode>);

    expect(screen.getByTestId("stream-visible")).toHaveTextContent("new spec words");
    expect(screen.getByTestId("stream-visible").parentElement).not.toHaveAttribute("data-streaming");
  });

  it("still streams after the pane remounts with new copy", () => {
    const { unmount } = render(renderStream("old summary", { memoryKey: "order-1:summary" }));
    unmount();

    render(renderStream("new summary words", { memoryKey: "order-1:summary" }));

    expect(screen.getByTestId("stream-visible")).toHaveTextContent("new");
    expect(screen.getByTestId("stream-visible")).not.toHaveTextContent("words");
  });
});
