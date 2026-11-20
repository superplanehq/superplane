import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactoryLine } from "@/api-client";

import { DeleteLineStageDialog } from "./DeleteLineStageDialog";

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

describe("DeleteLineStageDialog", () => {
  beforeEach(() => {
    updateLine.mockReset().mockResolvedValue({});
  });

  it("confirms and removes the selected optional stage", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <DeleteLineStageDialog
        organizationId="org-1"
        factoryId="factory-1"
        line={LINE}
        stage={{ id: "verify", key: "verify", name: "Verify", kind: "STAGE_KIND_VERIFY" }}
        onClose={onClose}
      />,
    );

    expect(screen.getByRole("heading", { name: "Delete Verify?" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Delete stage" }));

    await waitFor(() => expect(updateLine).toHaveBeenCalledTimes(1));
    expect(updateLine.mock.calls[0][0].stageConfiguration.stages.map((stage: { key: string }) => stage.key)).toEqual([
      "backlog",
      "done",
    ]);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
