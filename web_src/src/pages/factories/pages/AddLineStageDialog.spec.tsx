import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryLine } from "@/api-client";

import { AddLineStageDialog } from "./AddLineStageDialog";

const updateLine = vi.fn();

vi.mock("@/hooks/useFactoryData", () => ({
  useUpdateFactoryLine: () => ({ mutateAsync: updateLine, isPending: false }),
}));

const LINE: FactoriesFactoryLine = {
  id: "line-1",
  name: "Main",
  stages: [
    { id: "backlog", key: "backlog", name: "Backlog", kind: "STAGE_KIND_BACKLOG" },
    { id: "verify", key: "verify", name: "Verify", kind: "STAGE_KIND_VERIFY" },
    { id: "done", key: "done", name: "Done", kind: "STAGE_KIND_DONE" },
  ],
};

describe("AddLineStageDialog", () => {
  beforeEach(() => {
    updateLine.mockReset();
  });

  it("adds a named stage before Done and reports the saved stage", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    updateLine.mockImplementation(async ({ stageConfiguration }) => ({
      ...LINE,
      stages: stageConfiguration.stages.map((stage: { key: string }) =>
        stage.key.startsWith("new-")
          ? { id: "stage-qa", key: "custom-stage-qa", name: "QA", kind: "STAGE_KIND_CUSTOM" }
          : stage,
      ),
    }));

    render(
      <AddLineStageDialog
        organizationId="org-1"
        factoryId="factory-1"
        line={LINE}
        open
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    await user.type(screen.getByRole("textbox", { name: "Name" }), "QA");
    await user.click(screen.getByRole("button", { name: "Add stage" }));

    await waitFor(() => expect(onCreated).toHaveBeenCalledTimes(1));
    expect(updateLine.mock.calls[0][0].stageConfiguration.stages.map((stage: { name: string }) => stage.name)).toEqual([
      "Backlog",
      "Verify",
      "QA",
      "Done",
    ]);
    expect(onCreated).toHaveBeenCalledWith(expect.objectContaining({ name: "QA", key: "custom-stage-qa" }));
  });

  it("does not allow a duplicate stage name", async () => {
    const user = userEvent.setup();
    render(
      <AddLineStageDialog
        organizationId="org-1"
        factoryId="factory-1"
        line={LINE}
        open
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await user.type(screen.getByRole("textbox", { name: "Name" }), "verify");

    expect(screen.getByText("Stage names must be unique.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add stage" })).toBeDisabled();
  });
});
