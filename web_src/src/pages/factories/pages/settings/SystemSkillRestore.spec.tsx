import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { Toaster } from "sonner";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { CUSTOM_SYSTEM_SKILL, LATEST_SYSTEM_SKILL } from "../../__fixtures__/systemSkillFixtures";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

function renderEditor(fail = false) {
  return render(
    <>
      <Toaster />
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/agent/skills/${CUSTOM_SYSTEM_SKILL.id}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [{ ...CUSTOM_SYSTEM_SKILL }] },
          restoredAgentResourcesById: { [CUSTOM_SYSTEM_SKILL.id!]: LATEST_SYSTEM_SKILL },
          failAgentResourceRestoreForIds: fail ? [CUSTOM_SYSTEM_SKILL.id!] : [],
        }}
        experimentalFeatures={[FEATURE_WORKSPACE_SKILLS]}
      />
    </>,
  );
}

describe("system skill restore", () => {
  beforeAll(() => client.setConfig({ baseUrl: "http://localhost" }));
  it("requires confirmation and refreshes instructions and status after restoring", async () => {
    const user = userEvent.setup();
    renderEditor();
    expect(await screen.findByText("Custom", { exact: true }, { timeout: 8000 })).toBeInTheDocument();
    const markdown = screen.getByTestId("agent-resource-skill-markdown").querySelector("textarea")!;
    await user.type(markdown, "\nUnsaved edit");
    await user.click(screen.getByRole("button", { name: "Restore to latest" }));
    let dialog = screen.getByRole("alertdialog");
    expect(dialog).toHaveTextContent("Saved and unsaved edits will be removed.");
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(markdown.value).toContain("Unsaved edit");
    await user.click(screen.getByRole("button", { name: "Restore to latest" }));
    dialog = screen.getByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Restore to latest" }));
    expect(await screen.findByText("Latest", { exact: true })).toBeInTheDocument();
    await waitFor(() => expect(markdown.value).toContain("Standard refinement instructions."));
    expect(markdown.value).not.toContain("Unsaved edit");
    expect(screen.getByTestId("agent-resource-skill-command")).toHaveValue("/refine-task");
    expect(screen.getByTestId("skill-editor-enabled")).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Restore to latest" })).not.toBeInTheDocument();
  }, 15000);
  it("keeps custom instructions and the confirmation open when restore fails", async () => {
    const user = userEvent.setup();
    renderEditor(true);
    await screen.findByText("Custom", { exact: true }, { timeout: 8000 });
    const markdown = screen.getByTestId("agent-resource-skill-markdown").querySelector("textarea")!;
    await user.click(screen.getByRole("button", { name: "Restore to latest" }));
    await user.click(within(screen.getByRole("alertdialog")).getByRole("button", { name: "Restore to latest" }));
    expect(await screen.findByText("Could not restore skill.")).toBeInTheDocument();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(markdown.value).toContain("My custom instructions.");
    expect(screen.queryByText("Latest", { exact: true })).not.toBeInTheDocument();
  }, 15000);
});
