import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes, useNavigate } from "react-router";
import type { FactoriesFactoryAgentResource } from "@/api-client";
import { CUSTOM_SYSTEM_SKILL, LATEST_SYSTEM_SKILL } from "../../__fixtures__/systemSkillFixtures";
import { useSkillEditorPage } from "./useSkillEditorPage";

const otherSkill: FactoriesFactoryAgentResource = {
  id: "other-skill",
  name: "review-copy",
  kind: "KIND_SKILL",
  enabled: true,
  markdown: "---\nname: review-copy\ntitle: Review copy\n---\n\nReview the wording.",
};
let restoreResponse: Promise<FactoriesFactoryAgentResource>;
vi.mock("@/hooks/useFactoryAgentResources", () => ({
  useFactoryAgentResources: () => ({ data: [CUSTOM_SYSTEM_SKILL, otherSkill], isLoading: false }),
  useCreateFactoryAgentResource: () => ({ isPending: false }),
  useUpdateFactoryAgentResource: () => ({ isPending: false }),
  useDeleteFactoryAgentResource: () => ({ isPending: false }),
  useRestoreFactorySystemSkill: () => ({ isPending: false, mutateAsync: () => restoreResponse }),
}));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => true, isLoading: false }) }));
vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({ organizationId: "org", factoryId: "factory", factory: null }),
}));

function EditorHarness() {
  const editor = useSkillEditorPage();
  const navigate = useNavigate();
  return (
    <>
      <input aria-label="Skill name" value={editor.name} readOnly />
      <input aria-label="Command" value={editor.command} readOnly />
      <textarea
        aria-label="Instructions"
        value={editor.markdown}
        onChange={(event) => editor.setMarkdown(event.target.value)}
      />
      <button
        onClick={() => {
          editor.setPendingRestore(true);
          void editor.confirmRestore();
        }}
      >
        Restore
      </button>
      <button onClick={() => navigate(-1)}>Back</button>
      {editor.pendingRestore ? <p>Restore confirmation open</p> : null}
    </>
  );
}

describe("skill restore navigation", () => {
  it("keeps the current skill draft when a different skill's restore finishes", async () => {
    let finishRestore!: (resource: FactoriesFactoryAgentResource) => void;
    restoreResponse = new Promise((resolve) => {
      finishRestore = resolve;
    });
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/skills/other-skill", `/skills/${CUSTOM_SYSTEM_SKILL.id}`]} initialIndex={1}>
        <Routes>
          <Route path="/skills/:resourceId" element={<EditorHarness />} />
        </Routes>
      </MemoryRouter>,
    );
    await waitFor(() => expect(screen.getByLabelText("Command")).toHaveValue("/refine-task"));
    await user.click(screen.getByRole("button", { name: "Restore" }));
    await user.click(screen.getByRole("button", { name: "Back" }));
    await waitFor(() => expect(screen.getByLabelText("Command")).toHaveValue("/review-copy"));
    const instructions = screen.getByRole("textbox", { name: "Instructions" });
    await user.clear(instructions);
    await user.type(instructions, "Unsaved wording changes");
    await act(async () => {
      finishRestore(LATEST_SYSTEM_SKILL);
      await restoreResponse;
    });
    expect(screen.getByLabelText("Skill name")).toHaveValue("Review copy");
    expect(screen.getByLabelText("Command")).toHaveValue("/review-copy");
    expect(instructions).toHaveValue("Unsaved wording changes");
    expect(screen.queryByText("Restore confirmation open")).not.toBeInTheDocument();
  });
});
