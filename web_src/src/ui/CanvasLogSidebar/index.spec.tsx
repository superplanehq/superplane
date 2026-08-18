import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CanvasLogSidebar } from "./index";

vi.mock("@/pages/app/ErrorsConsoleContent", () => ({
  ErrorsConsoleContent: () => <div data-testid="errors-console-content" />,
}));

describe("CanvasLogSidebar", () => {
  it("insets from the right so it stays beside absolute right panels", () => {
    render(
      <CanvasLogSidebar
        isOpen
        onClose={() => undefined}
        rightOffset={380}
        searchValue=""
        onSearchChange={() => undefined}
        entries={[]}
        counts={{ total: 0, error: 0, warning: 0, success: 0 }}
      />,
    );

    expect(screen.getByTestId("canvas-log-sidebar")).toHaveStyle({ right: "380px" });
  });

  it("spans the full canvas width when no right offset is set", () => {
    render(
      <CanvasLogSidebar
        isOpen
        onClose={() => undefined}
        searchValue=""
        onSearchChange={() => undefined}
        entries={[]}
        counts={{ total: 0, error: 0, warning: 0, success: 0 }}
      />,
    );

    expect(screen.getByTestId("canvas-log-sidebar")).toHaveStyle({ right: "0px" });
  });
});
