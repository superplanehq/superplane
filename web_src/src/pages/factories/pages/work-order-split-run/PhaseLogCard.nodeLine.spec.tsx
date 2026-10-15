import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import sentryIcon from "@/assets/icons/integrations/sentry.svg";

import { PhaseLogCard } from "./PhaseLogCard";
import { idleLiveLogStream, line, PHASE } from "./PhaseLogCard.testHelpers";
import type { SplitRunStreamLine } from "./splitRunMocks";

const useLiveLogStreamMock = vi.fn();

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

beforeEach(() => {
  useLiveLogStreamMock.mockReturnValue(idleLiveLogStream(vi.fn()));
});

describe("PhaseLogCard node line", () => {
  const NODE_STREAM: SplitRunStreamLine[] = [
    line({
      id: "planner-agent",
      componentName: "Agent - Plan for GH Issue",
      componentType: "Run Claude Code",
      at: "12:24:02",
      duration: "1m 20s",
    }),
  ];

  it("shows the node name, accessible status, and formatted duration", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={NODE_STREAM} />);

    const row = screen.getByTestId("split-run-stream-line-planner-agent");
    expect(within(row).getByText("Agent - Plan for GH Issue")).toBeInTheDocument();
    expect(within(row).queryByText("passed")).not.toBeInTheDocument();
    expect(within(row).queryByText(">")).not.toBeInTheDocument();
    expect(within(row).queryByText("12:24:02")).not.toBeInTheDocument();
    expect(within(row).queryByText("Run Claude Code")).not.toBeInTheDocument();

    const statusTime = within(row).getByTestId("split-run-stream-duration-planner-agent");
    expect(statusTime).toHaveAccessibleName("Passed");
    expect(statusTime).toHaveTextContent("✓");
    expect(statusTime).toHaveTextContent("01:20");
    expect(statusTime).not.toHaveTextContent("Passed");
  });

  it("shows the artifact produced by a stream node", () => {
    render(
      <PhaseLogCard
        phase={PHASE}
        expanded
        stream={[
          line({
            id: "planner-agent",
            componentName: "Agent - Plan for GH Issue",
            artifact: {
              id: "art-plan",
              type: "TYPE_MARKDOWN",
              data: { name: "PLAN.md", title: "PLAN.md" },
            },
          }),
        ]}
      />,
    );

    const row = screen.getByTestId("split-run-stream-line-planner-agent");
    expect(within(row).getByRole("button", { name: "PLAN.md" })).toBeInTheDocument();
  });

  it("inverts a Sentry stream-line logo in dark mode", () => {
    render(
      <PhaseLogCard
        phase={PHASE}
        expanded
        stream={[
          line({
            id: "on-issue",
            componentName: "On Issue",
            iconSrc: sentryIcon,
          }),
        ]}
      />,
    );

    const icon = screen.getByTestId("split-run-stream-line-on-issue").querySelector("img");
    expect(icon).toHaveAttribute("src", sentryIcon);
    expect(icon?.className).toContain("dark:brightness-0");
    expect(icon?.className).toContain("dark:invert");
  });
});
