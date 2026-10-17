import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes, useNavigate } from "react-router";
import type { FactoriesFactoryAgentResource } from "@/api-client";
import { CUSTOM_SYSTEM_SKILL, LATEST_SYSTEM_SKILL } from "../../__fixtures__/systemSkillFixtures";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useSkillEditorPage } from "./useSkillEditorPage";

const otherSkill: FactoriesFactoryAgentResource = {
  id: "other-skill",
  name: "review-copy",
  kind: "KIND_SKILL",
  enabled: true,
  markdown: "---\nname: review-copy\ntitle: Review copy\n---\n\nReview the wording.",
};
let restoreResponse: Promise<FactoriesFactoryAgentResource>;
const createSkill = vi.fn((_input: unknown) => Promise.resolve({}));
vi.mock("@/hooks/useFactoryAgentResources", () => ({
  useFactoryAgentResources: () => ({ data: [CUSTOM_SYSTEM_SKILL, otherSkill], isLoading: false }),
  useCreateFactoryAgentResource: () => ({ isPending: false, mutateAsync: createSkill }),
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
      <Input aria-label="Skill name" value={editor.name} onChange={(event) => editor.setName(event.target.value)} />
      <Input aria-label="Command" value={editor.command} readOnly />
      <Textarea
        aria-label="Instructions"
        value={editor.markdown}
        onChange={(event) => editor.setMarkdown(event.target.value)}
      />
      <Button
        onClick={() => {
          editor.setPendingRestore(true);
          void editor.confirmRestore();
        }}
      >
        Restore
      </Button>
      <Button onClick={() => navigate(-1)}>Back</Button>
      <Button onClick={() => void editor.handleSave()}>Save</Button>
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

const PASTED_SKILL = "---\nname: ponytail-review\ntitle: \ndescription: \n---\n\nReview the diff.";

function renderNewSkillEditor() {
  render(
    <MemoryRouter initialEntries={["/skills/new"]}>
      <Routes>
        <Route path="/skills/:resourceId" element={<EditorHarness />} />
      </Routes>
    </MemoryRouter>,
  );
  return {
    user: userEvent.setup(),
    nameInput: screen.getByLabelText("Skill name"),
    commandInput: screen.getByLabelText("Command"),
    instructions: screen.getByRole("textbox", { name: "Instructions" }) as HTMLTextAreaElement,
  };
}

describe("skill Name and SKILL.md sync", () => {
  it("fills the Name field and command when SKILL.md with a name is pasted", async () => {
    const { user, nameInput, commandInput, instructions } = renderNewSkillEditor();
    await user.click(instructions);
    await user.clear(instructions);
    await user.paste(PASTED_SKILL);
    expect(nameInput).toHaveValue("ponytail-review");
    expect(commandInput).toHaveValue("/ponytail-review");
  });

  it("writes typed Name edits into the frontmatter", async () => {
    const { user, nameInput, instructions } = renderNewSkillEditor();
    await user.type(nameInput, "Oy Pirate");
    expect(instructions.value).toContain("name: oypirate");
    expect(instructions.value).toContain("title: Oy Pirate");
  });

  it("updates Name and command when the frontmatter is edited after a typed Name", async () => {
    const { user, nameInput, commandInput, instructions } = renderNewSkillEditor();
    await user.type(nameInput, "Oy Pirate");
    await user.clear(instructions);
    await user.paste(PASTED_SKILL);
    expect(nameInput).toHaveValue("ponytail-review");
    expect(commandInput).toHaveValue("/ponytail-review");
  });

  it("keeps the typed Name when a markdown edit has no name or title", async () => {
    const { user, nameInput, commandInput, instructions } = renderNewSkillEditor();
    await user.type(nameInput, "Oy Pirate");
    await user.clear(instructions);
    await user.paste("Ahoy, no frontmatter here.");
    expect(nameInput).toHaveValue("Oy Pirate");
    expect(commandInput).toHaveValue("/oypirate");
  });

  it("saves the pasted name instead of the Name placeholder", async () => {
    createSkill.mockClear();
    const { user, instructions } = renderNewSkillEditor();
    await user.click(instructions);
    await user.clear(instructions);
    await user.paste(PASTED_SKILL);
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(createSkill).toHaveBeenCalledTimes(1));
    expect(createSkill.mock.calls[0][0]).toMatchObject({
      name: "ponytail-review",
      markdown: expect.stringContaining("name: ponytail-review"),
    });
  });
});
