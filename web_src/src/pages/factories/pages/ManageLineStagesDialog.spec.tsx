import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryLine } from "@/api-client";

import { ManageLineStagesDialog } from "./ManageLineStagesDialog";

const updateLine = vi.fn();

vi.mock("@/hooks/useFactoryData", () => ({
  useUpdateFactoryLine: () => ({ mutateAsync: updateLine, isPending: false }),
}));

const LINE: FactoriesFactoryLine = {
  id: "line-1",
  name: "Main",
  columnColors: { "custom-review": "teal" },
  stages: [
    { id: "stage-backlog", key: "backlog", name: "Backlog", kind: "STAGE_KIND_BACKLOG" },
    { id: "stage-phase", key: "phase-0", name: "Implement", kind: "STAGE_KIND_PHASE", stepIndex: 0 },
    { id: "stage-review", key: "custom-review", name: "Review", kind: "STAGE_KIND_CUSTOM" },
    { id: "stage-verify", key: "verify", name: "Verify", kind: "STAGE_KIND_VERIFY" },
    { id: "stage-done", key: "done", name: "Done", kind: "STAGE_KIND_DONE" },
  ],
};

function renderDialog(overrides: Partial<React.ComponentProps<typeof ManageLineStagesDialog>> = {}) {
  const onClose = vi.fn();
  render(
    <ManageLineStagesDialog
      organizationId="org-1"
      factoryId="factory-1"
      line={LINE}
      addStageOnOpen={false}
      onClose={onClose}
      {...overrides}
    />,
  );
  return { onClose };
}

describe("ManageLineStagesDialog", () => {
  beforeEach(() => {
    updateLine.mockReset().mockResolvedValue({});
  });

  it("saves a renamed, reordered, and recolored custom stage", async () => {
    const user = userEvent.setup();
    const { onClose } = renderDialog();

    await user.clear(screen.getByRole("textbox", { name: "Review name" }));
    await user.type(screen.getByRole("textbox", { name: "New stage name" }), "QA review");
    await user.click(screen.getByRole("button", { name: "Move QA review up" }));
    await user.click(screen.getByRole("combobox", { name: "QA review color" }));
    await user.click(screen.getByRole("option", { name: "Purple" }));
    await user.click(screen.getByRole("button", { name: "Save stages" }));

    await waitFor(() => expect(updateLine).toHaveBeenCalledTimes(1));
    const request = updateLine.mock.calls[0][0];
    expect(request.lineId).toBe("line-1");
    expect(request.stageConfiguration.stages.map((stage: { key: string }) => stage.key)).toEqual([
      "backlog",
      "custom-review",
      "phase-0",
      "verify",
      "done",
    ]);
    expect(request.stageConfiguration.stages[1].name).toBe("QA review");
    expect(request.columnColors["custom-review"]).toBe("purple");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("adds and removes custom stages while system stages stay locked", async () => {
    const user = userEvent.setup();
    renderDialog({ addStageOnOpen: true });

    expect(screen.getAllByTitle("System stage")).toHaveLength(4);
    await user.click(screen.getByRole("button", { name: "Remove Review" }));
    const emptyName = screen.getByRole("textbox", { name: "New stage name" });
    await user.type(emptyName, "Approval");
    await user.click(screen.getByRole("button", { name: "Save stages" }));

    await waitFor(() => expect(updateLine).toHaveBeenCalledTimes(1));
    const custom = updateLine.mock.calls[0][0].stageConfiguration.stages.filter(
      (stage: { kind: string }) => stage.kind === "STAGE_KIND_CUSTOM",
    );
    expect(custom).toHaveLength(1);
    expect(custom[0]).toMatchObject({ id: "", name: "Approval", kind: "STAGE_KIND_CUSTOM" });
    expect(custom[0].key.startsWith("new-")).toBe(true);
  });

  it("blocks duplicate names and confirms before discarding changes", async () => {
    const user = userEvent.setup();
    const { onClose } = renderDialog();
    const review = screen.getByRole("textbox", { name: "Review name" });

    await user.clear(review);
    await user.type(screen.getByRole("textbox", { name: "New stage name" }), "Verify");
    expect(screen.getByRole("alert")).toHaveTextContent("Stage names must be unique.");
    expect(screen.getByRole("button", { name: "Save stages" })).toBeDisabled();

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    const confirmation = screen.getByRole("alertdialog");
    expect(within(confirmation).getByText("Discard stage changes?")).toBeInTheDocument();
    await user.click(within(confirmation).getByRole("button", { name: "Discard changes" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
