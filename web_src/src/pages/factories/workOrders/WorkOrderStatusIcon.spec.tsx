import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { WorkOrderStatusIcon } from "./WorkOrderStatusIcon";

describe("WorkOrderStatusIcon", () => {
  it("uses a spinning ring for running and no ping disk", () => {
    const { container } = render(<WorkOrderStatusIcon status="running" title="Running" />);

    expect(container.querySelector(".animate-spin")).not.toBeNull();
    expect(container.querySelector(".animate-ping")).toBeNull();
    expect(container.querySelector(".lucide-loader-circle")).toBeNull();
    expect(container.querySelector("[class*='--status-running-dot']")).not.toBeNull();
  });

  it("uses a distinct mark for each other status", () => {
    const { rerender, container } = render(<WorkOrderStatusIcon status="completed" title="Completed" />);
    expect(container.querySelector("[data-status-mark='completed']")).not.toBeNull();

    rerender(<WorkOrderStatusIcon status="failed" title="Failed" />);
    expect(container.querySelector("[data-status-mark='failed']")).not.toBeNull();

    rerender(<WorkOrderStatusIcon status="waiting" title="Needs attention" />);
    expect(container.querySelector("[data-status-mark='waiting']")).not.toBeNull();

    rerender(<WorkOrderStatusIcon status="draft" title="Draft" />);
    expect(container.querySelector("[data-status-mark='draft']")).not.toBeNull();

    rerender(<WorkOrderStatusIcon status="rejected" title="Rejected" />);
    expect(container.querySelector("[data-status-mark='rejected']")).not.toBeNull();

    rerender(<WorkOrderStatusIcon status="cancelled" title="Canceled" />);
    expect(container.querySelector("[data-status-mark='cancelled']")).not.toBeNull();
  });

  it("fills attention and closed states so the color disk is easy to scan", () => {
    const { rerender, container } = render(<WorkOrderStatusIcon status="completed" title="Completed" />);
    expect(container.querySelector("[data-status-mark='completed']")?.className).toContain("--status-completed-dot");
    expect(container.querySelector("svg[data-status-glyph]")).not.toBeNull();

    rerender(<WorkOrderStatusIcon status="failed" title="Failed" />);
    expect(container.querySelector("[data-status-mark='failed']")?.className).toContain("--status-failed-dot");

    rerender(<WorkOrderStatusIcon status="waiting" title="Needs attention" />);
    expect(container.querySelector("[data-status-mark='waiting']")?.className).toContain("--status-waiting-dot");
  });
});
