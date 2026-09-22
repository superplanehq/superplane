import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "bun:test";

import { resetFactoryBoardLaneScrollPositions, useFactoryBoardLaneScroll } from "@/hooks/useFactoryBoardLaneScroll";

function Lane({ persistenceKey }: { persistenceKey: string }) {
  const { scrollRef, handleScroll } = useFactoryBoardLaneScroll(persistenceKey);
  return (
    <ul data-testid="lane" ref={scrollRef} onScroll={handleScroll}>
      <li>content</li>
    </ul>
  );
}

describe("useFactoryBoardLaneScroll", () => {
  beforeEach(() => {
    resetFactoryBoardLaneScrollPositions();
    Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, value: 2000 });
    Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, value: 240 });
  });

  afterEach(() => {
    delete (HTMLElement.prototype as { scrollHeight?: number }).scrollHeight;
    delete (HTMLElement.prototype as { clientHeight?: number }).clientHeight;
  });

  it("resets scroll when the same element receives a new key with no saved position", () => {
    const { rerender } = render(<Lane persistenceKey="ws:line-a:step-0" />);
    const lane = screen.getByTestId("lane");
    lane.scrollTop = 1760;
    fireEvent.scroll(lane);

    rerender(<Lane persistenceKey="ws:line-b:step-0" />);

    expect(screen.getByTestId("lane").scrollTop).toBe(0);
  });

  it("restores the previous key after a switch without writing the new scroll onto it", () => {
    const { rerender } = render(<Lane persistenceKey="ws:line-a:step-0" />);
    const first = screen.getByTestId("lane");
    first.scrollTop = 1760;
    fireEvent.scroll(first);

    rerender(<Lane persistenceKey="ws:line-b:step-0" />);
    const second = screen.getByTestId("lane");
    second.scrollTop = 900;
    fireEvent.scroll(second);

    rerender(<Lane persistenceKey="ws:line-a:step-0" />);

    expect(screen.getByTestId("lane").scrollTop).toBe(1760);
  });

  it("restores a saved key without writing that value onto the previous key", () => {
    const { unmount } = render(<Lane persistenceKey="ws:line-a:step-0" />);
    const first = screen.getByTestId("lane");
    first.scrollTop = 1760;
    fireEvent.scroll(first);
    unmount();

    const secondView = render(<Lane persistenceKey="ws:line-b:step-0" />);
    const second = screen.getByTestId("lane");
    second.scrollTop = 900;
    fireEvent.scroll(second);
    secondView.unmount();

    const { rerender: rerenderShared } = render(<Lane persistenceKey="ws:line-a:step-0" />);
    expect(screen.getByTestId("lane").scrollTop).toBe(1760);

    rerenderShared(<Lane persistenceKey="ws:line-b:step-0" />);
    expect(screen.getByTestId("lane").scrollTop).toBe(900);

    rerenderShared(<Lane persistenceKey="ws:line-a:step-0" />);
    expect(screen.getByTestId("lane").scrollTop).toBe(1760);
  });
});
