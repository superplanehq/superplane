import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CommitStagingDialog } from "./CommitStagingDialog";

describe("CommitStagingDialog", () => {
  it("keeps long unbroken commit messages wrapping inside the input", () => {
    render(<CommitStagingDialog open onOpenChange={vi.fn()} onCommit={vi.fn()} />);

    const input = screen.getByTestId("canvas-commit-message-input");

    // `wrap-anywhere` (overflow-wrap: anywhere) lets a space-less message wrap in
    // place and, unlike `break-words`, shrinks the textarea's intrinsic width so a
    // `field-sizing: content` textarea can no longer overflow the dialog (#6168).
    expect(input.className).toContain("wrap-anywhere");
    expect(input.className).not.toContain("break-words");
  });
});
