import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import type { FactoriesFactoryAgentResource } from "@/api-client";
import { skillFrontmatterField } from "./skillFrontmatter";
import { useSkillEditorPage } from "./useSkillEditorPage";

const existingSkill: FactoriesFactoryAgentResource = {
  id: "skill-review",
  name: "review-copy",
  kind: "KIND_SKILL",
  enabled: true,
  markdown: "---\nname: review-copy\ntitle: Review copy\n---\n\nReview the wording.",
};

const divergentSkill: FactoriesFactoryAgentResource = {
  id: "skill-divergent",
  name: "review-copy",
  kind: "KIND_SKILL",
  enabled: true,
  markdown: "---\nname: ponytail-review\ntitle: Review copy\n---\n\nReview the wording.",
};

const { createSkill, updateSkill } = vi.hoisted(() => ({
  createSkill: vi.fn((_body: { kind: string; name: string; enabled: boolean; markdown: string }) =>
    Promise.resolve({}),
  ),
  updateSkill: vi.fn((_input: { resourceId: string; name?: string; markdown?: string; enabled?: boolean }) =>
    Promise.resolve({}),
  ),
}));

vi.mock("@/hooks/useFactoryAgentResources", () => ({
  useFactoryAgentResources: () => ({ data: [existingSkill, divergentSkill], isLoading: false }),
  useCreateFactoryAgentResource: () => ({ isPending: false, mutateAsync: createSkill }),
  useUpdateFactoryAgentResource: () => ({ isPending: false, mutateAsync: updateSkill }),
  useDeleteFactoryAgentResource: () => ({ isPending: false }),
  useRestoreFactorySystemSkill: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ canAct: () => true, isLoading: false }) }));
vi.mock("../../layout/factoriesLayoutContext", () => ({
  useFactoriesLayout: () => ({ organizationId: "org", factoryId: "factory", factory: null }),
}));
vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

function renderSkillEditor(path = "/skills/new") {
  return renderHook(() => useSkillEditorPage(), {
    wrapper: ({ children }: { children: ReactNode }) => (
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/skills/:resourceId" element={children} />
        </Routes>
      </MemoryRouter>
    ),
  });
}

describe("skill name sync", () => {
  beforeEach(() => {
    createSkill.mockReset().mockResolvedValue({});
    updateSkill.mockReset().mockResolvedValue({});
  });

  it("fills the name and command from a pasted file name", () => {
    const { result } = renderSkillEditor();

    act(() => {
      result.current.setMarkdown("---\nname: ponytail-review\n---\n\nCheck the diff.\n");
    });

    expect(result.current.name).toBe("ponytail-review");
    expect(result.current.command).toBe("/ponytail-review");
  });

  it("keeps the file command when the pasted title differs from the name", () => {
    const { result } = renderSkillEditor();

    act(() => {
      result.current.setMarkdown("---\nname: ponytail-review\ntitle: Ponytail Review\n---\n\nCheck the diff.\n");
    });

    expect(result.current.name).toBe("Ponytail Review");
    expect(result.current.command).toBe("/ponytail-review");
  });

  it("builds the command from the title when the file name is blank", () => {
    const { result } = renderSkillEditor();

    act(() => {
      result.current.setMarkdown("---\nname: \ntitle: Ponytail Review\n---\n\nCheck the diff.\n");
    });

    expect(result.current.name).toBe("Ponytail Review");
    expect(result.current.command).toBe("/ponytailreview");
  });

  it("keeps a typed name when only the body changes", () => {
    const { result } = renderSkillEditor();

    act(() => {
      result.current.setName("Review copy");
    });
    act(() => {
      result.current.setMarkdown(`${result.current.markdown}\nKeep this body.`);
    });

    expect(result.current.name).toBe("Review copy");
    expect(result.current.command).toBe("/reviewcopy");
    expect(skillFrontmatterField(result.current.markdown, "name")).toBe("reviewcopy");
    expect(skillFrontmatterField(result.current.markdown, "title")).toBe("Review copy");
  });

  it("keeps a typed name when the file name and title are blank", () => {
    const { result } = renderSkillEditor();

    act(() => {
      result.current.setName("Review copy");
    });
    act(() => {
      result.current.setMarkdown("---\nname: \ntitle: \n---\n\nOnly the body.\n");
    });

    expect(result.current.name).toBe("Review copy");
    expect(result.current.command).toBe("/reviewcopy");
  });

  it("follows a pasted name after the user typed a different name", () => {
    const { result } = renderSkillEditor();

    act(() => {
      result.current.setName("Review copy");
    });
    act(() => {
      result.current.setMarkdown("---\nname: ponytail-review\n---\n\nPasted.\n");
    });

    expect(result.current.name).toBe("ponytail-review");
    expect(result.current.command).toBe("/ponytail-review");
  });

  it("keeps a YAML comment out of the pasted command and the saved name", async () => {
    const { result } = renderSkillEditor();
    act(() => {
      result.current.setMarkdown("---\nname: review-copy # command name\n---\n\nCheck the diff.\n");
    });

    expect(result.current.name).toBe("review-copy");
    expect(result.current.command).toBe("/review-copy");

    await act(async () => {
      await result.current.handleSave();
    });

    const saved = createSkill.mock.calls[0]?.[0];
    expect(saved?.name).toBe("review-copy");
    expect(skillFrontmatterField(saved?.markdown ?? "", "name")).toBe("review-copy");
  });

  it("saves the pasted name instead of an empty name", async () => {
    const { result } = renderSkillEditor();
    act(() => {
      result.current.setMarkdown("---\nname: ponytail-review\ndescription: Review the diff.\n---\n\nCheck the diff.\n");
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(createSkill).toHaveBeenCalledTimes(1);
    const saved = createSkill.mock.calls[0]?.[0];
    expect(saved?.name).toBe("ponytail-review");
    expect(skillFrontmatterField(saved?.markdown ?? "", "name")).toBe("ponytail-review");
    expect(skillFrontmatterField(saved?.markdown ?? "", "title")).toBe("ponytail-review");
    expect(updateSkill).not.toHaveBeenCalled();
  });

  it("does not save a reserved command from the file", async () => {
    const { result } = renderSkillEditor();
    act(() => {
      result.current.setMarkdown("---\nname: setup\n---\n\nDo not save this name.\n");
    });

    await act(async () => {
      await result.current.handleSave();
    });

    expect(result.current.nameError).toBe("The name setup is reserved.");
    expect(createSkill).not.toHaveBeenCalled();
    expect(updateSkill).not.toHaveBeenCalled();
  });

  it("updates name and command when an existing skill file changes", () => {
    const { result } = renderSkillEditor(`/skills/${existingSkill.id}`);

    expect(result.current.name).toBe("Review copy");
    expect(result.current.command).toBe("/review-copy");

    act(() => {
      result.current.setMarkdown("---\nname: ponytail-review\ntitle: Ponytail Review\n---\n\nEdited.\n");
    });

    expect(result.current.name).toBe("Ponytail Review");
    expect(result.current.command).toBe("/ponytail-review");
  });

  it("keeps the loaded command when only the body of an existing skill changes", () => {
    const { result } = renderSkillEditor(`/skills/${divergentSkill.id}`);

    expect(result.current.name).toBe("Review copy");
    expect(result.current.command).toBe("/review-copy");

    act(() => {
      result.current.setMarkdown(`${divergentSkill.markdown}\nMore wording.`);
    });

    expect(result.current.name).toBe("Review copy");
    expect(result.current.command).toBe("/review-copy");
  });
});
