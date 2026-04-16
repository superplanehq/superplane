import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { ChainItem, type ChainItemData } from "./ChainItem";

describe("ChainItem details errors", () => {
  it("renders error messages fully readable without changing normal detail row truncation", () => {
    const errorMessage =
      "The remote API returned a validation error after processing the release payload. Please review the branch protection checks and retry the deployment.";

    const originalExecution: CanvasesCanvasNodeExecution = {
      createdAt: "2026-04-15T10:00:00Z",
      updatedAt: "2026-04-15T10:00:30Z",
      state: "STATE_FINISHED",
      resultReason: "RESULT_REASON_ERROR",
      resultMessage: errorMessage,
    };

    const item: ChainItemData = {
      id: "chain-item-1",
      nodeId: "node-1",
      componentName: "Deploy",
      nodeName: "Deploy to production",
      nodeIcon: "box",
      state: "failed",
      originalExecution,
      tabData: {
        current: {
          branch: "release/2026.04",
        },
      },
    };

    render(<ChainItem item={item} index={0} totalItems={1} isOpen={true} onToggleOpen={() => {}} />);

    const errorValue = screen.getByText(errorMessage);
    expect(errorValue).toBeInTheDocument();
    expect(errorValue).toHaveClass("whitespace-pre-wrap", "break-words", "select-text");
    expect(errorValue.className).not.toMatch(/\btruncate\b/);

    const detailValue = screen.getByText("release/2026.04");
    expect(detailValue).toHaveClass("truncate");
  });
});
