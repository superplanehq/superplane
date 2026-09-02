import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { useFollowLogScroll } from "./useFollowLogScroll";

function mockOverflow(el: HTMLElement, box: { height: number; view: number }) {
  Object.defineProperty(el, "scrollHeight", { configurable: true, get: () => box.height });
  Object.defineProperty(el, "clientHeight", { configurable: true, get: () => box.view });
}

function FollowLog({
  tick,
  running = "implement",
  resumeOnBottom = false,
}: {
  tick: number;
  running?: string | null;
  resumeOnBottom?: boolean;
}) {
  const follow = useFollowLogScroll(running, tick, { resumeOnBottom });
  return (
    <>
      <span data-testid="following">{follow.following ? "on" : "off"}</span>
      <button type="button" onClick={() => follow.setFollowing(false)}>
        Stop follow
      </button>
      <ol ref={follow.scrollRef} onScroll={follow.onScroll} data-testid="log-scroller">
        <li>start</li>
      </ol>
    </>
  );
}

async function settleScrollIgnore() {
  await new Promise<void>((resolve) => {
    requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
  });
}

async function appendNestedLine(scroller: HTMLElement, box: { height: number; view: number }, height: number) {
  box.height = height;
  const line = document.createElement("li");
  line.textContent = "new command";
  scroller.appendChild(line);
}

describe("useFollowLogScroll", () => {
  it("keeps the scroller at the bottom when nested log lines appear", async () => {
    const box = { height: 200, view: 100 };
    render(<FollowLog tick={1} />);
    const scroller = screen.getByTestId("log-scroller");
    mockOverflow(scroller, box);
    scroller.scrollTop = 100;

    await appendNestedLine(scroller, box, 400);

    await waitFor(() => {
      expect(scroller.scrollTop).toBe(400);
    });
  });

  it("does not move the scroller when Follow is off", async () => {
    const box = { height: 200, view: 100 };
    const user = userEvent.setup();
    render(<FollowLog tick={1} />);
    const scroller = screen.getByTestId("log-scroller");
    mockOverflow(scroller, box);

    await user.click(screen.getByRole("button", { name: "Stop follow" }));
    scroller.scrollTop = 40;

    await appendNestedLine(scroller, box, 400);

    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(scroller.scrollTop).toBe(40);
  });

  it("turns Follow back on at the bottom when resumeOnBottom is on", async () => {
    const box = { height: 400, view: 100 };
    render(<FollowLog tick={1} resumeOnBottom />);
    const scroller = screen.getByTestId("log-scroller");
    mockOverflow(scroller, box);
    await settleScrollIgnore();

    scroller.scrollTop = 0;
    fireEvent.scroll(scroller);
    expect(screen.getByTestId("following")).toHaveTextContent("off");

    scroller.scrollTop = 300;
    fireEvent.scroll(scroller);
    expect(screen.getByTestId("following")).toHaveTextContent("on");
  });

  it("does not turn Follow back on at the bottom by default", async () => {
    const box = { height: 400, view: 100 };
    render(<FollowLog tick={1} />);
    const scroller = screen.getByTestId("log-scroller");
    mockOverflow(scroller, box);
    await settleScrollIgnore();

    scroller.scrollTop = 0;
    fireEvent.scroll(scroller);
    expect(screen.getByTestId("following")).toHaveTextContent("off");

    scroller.scrollTop = 300;
    fireEvent.scroll(scroller);
    expect(screen.getByTestId("following")).toHaveTextContent("off");
  });

  it("does not pin the scroller when only text inside a line changes", async () => {
    const box = { height: 200, view: 100 };
    render(<FollowLog tick={1} />);
    const scroller = screen.getByTestId("log-scroller");
    mockOverflow(scroller, box);
    scroller.scrollTop = 100;
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
    });
    const settledTop = scroller.scrollTop;

    box.height = 400;
    const line = scroller.querySelector("li");
    expect(line?.firstChild?.nodeType).toBe(Node.TEXT_NODE);
    line!.firstChild!.nodeValue = "Running 00:01";

    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(scroller.scrollTop).toBe(settledTop);
  });
});
