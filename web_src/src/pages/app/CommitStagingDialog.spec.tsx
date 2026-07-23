import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CommitStagingDialog } from "./CommitStagingDialog";

describe("CommitStagingDialog", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  const renderDialog = (props?: Partial<React.ComponentProps<typeof CommitStagingDialog>>) => {
    const onCommit = vi.fn();
    const onOpenChange = vi.fn();
    render(<CommitStagingDialog open onOpenChange={onOpenChange} onCommit={onCommit} {...props} />);
    return { onCommit, onOpenChange };
  };

  it("submits on Enter with a non-empty message", async () => {
    const user = userEvent.setup();
    const { onCommit } = renderDialog();

    const input = screen.getByTestId("canvas-commit-message-input");
    await user.type(input, "Update triggers{Enter}");

    expect(onCommit).toHaveBeenCalledTimes(1);
    expect(onCommit).toHaveBeenCalledWith("Update triggers");
  });

  it("submits on Cmd+Enter", async () => {
    const user = userEvent.setup();
    const { onCommit } = renderDialog();

    const input = screen.getByTestId("canvas-commit-message-input");
    await user.type(input, "Update triggers");
    await user.keyboard("{Meta>}{Enter}{/Meta}");

    expect(onCommit).toHaveBeenCalledTimes(1);
    expect(onCommit).toHaveBeenCalledWith("Update triggers");
  });

  it("inserts a newline on Shift+Enter instead of submitting", async () => {
    const user = userEvent.setup();
    const { onCommit } = renderDialog();

    const input = screen.getByTestId("canvas-commit-message-input");
    await user.type(input, "line one{Shift>}{Enter}{/Shift}line two");

    expect(onCommit).not.toHaveBeenCalled();
    expect((input as HTMLTextAreaElement).value).toBe("line one\nline two");
  });

  it("does not submit an empty message on Enter", async () => {
    const user = userEvent.setup();
    const { onCommit } = renderDialog();

    const input = screen.getByTestId("canvas-commit-message-input");
    input.focus();
    await user.keyboard("{Enter}");

    expect(onCommit).not.toHaveBeenCalled();
  });

  it("does not submit while pending", async () => {
    const user = userEvent.setup();
    const { onCommit } = renderDialog({ pending: true });

    const input = screen.getByTestId("canvas-commit-message-input");
    input.focus();
    await user.keyboard("{Enter}");

    expect(onCommit).not.toHaveBeenCalled();
  });
});
