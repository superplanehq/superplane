import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import type { LineBoardStage } from "../lib/lineBoardStages";
import { AddLineStageDialog } from "./AddLineStageDialog";

const STAGES: LineBoardStage[] = [
  { id: "backlog", key: "backlog", name: "Backlog", kind: "STAGE_KIND_BACKLOG" },
  { id: "verify", key: "verify", name: "Verify", kind: "STAGE_KIND_PHASE", autoAdvance: false },
  { id: "done", key: "done", name: "Done", kind: "STAGE_KIND_DONE" },
];

describe("AddLineStageDialog", () => {
  it("reports the trimmed name without saving", async () => {
    const user = userEvent.setup();
    const onAdd = vi.fn();
    render(<AddLineStageDialog stages={STAGES} open onClose={vi.fn()} onAdd={onAdd} />);

    await user.type(screen.getByRole("textbox", { name: "Name" }), "  QA {Enter}");

    expect(onAdd).toHaveBeenCalledWith("QA");
  });

  it("does not allow a duplicate stage name", async () => {
    const user = userEvent.setup();
    render(<AddLineStageDialog stages={STAGES} open onClose={vi.fn()} onAdd={vi.fn()} />);

    await user.type(screen.getByRole("textbox", { name: "Name" }), "verify");

    expect(screen.getByText("Stage names must be unique.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add stage" })).toBeDisabled();
  });
});
