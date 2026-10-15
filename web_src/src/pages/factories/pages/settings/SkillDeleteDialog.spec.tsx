import { render, screen, within, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";
import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { CUSTOM_SYSTEM_SKILL } from "../../__fixtures__/systemSkillFixtures";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value }: { value?: string }) => <textarea value={value ?? ""} readOnly />,
}));

function renderEditor(system = false, fail = false, inUse = true) {
  render(
    <FactoriesHarness
      pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/agent/skills/${CUSTOM_SYSTEM_SKILL.id}`}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        agentResourcesByFactoryId: {
          [PRIMARY_FACTORY_ID]: [
            { ...CUSTOM_SYSTEM_SKILL, systemSkill: system ? CUSTOM_SYSTEM_SKILL.systemSkill : undefined },
          ],
        },
        skillUsageByName: { "refine-task": inUse ? [{ id: "review-automation", name: "Review changes" }] : [] },
        failSkillUsage: fail,
      }}
      experimentalFeatures={[FEATURE_WORKSPACE_SKILLS]}
    />,
  );
}

describe("skill deletion warning", () => {
  beforeAll(() => client.setConfig({ baseUrl: "http://localhost" }));
  for (const system of [false, true]) {
    it(`shows prompt usage before deletion of a ${system ? "system" : "user"} skill`, async () => {
      const user = userEvent.setup();
      renderEditor(system);
      await user.click(await screen.findByRole("button", { name: "Delete" }));
      const dialog = await screen.findByRole("alertdialog");
      expect(await within(dialog).findByRole("link", { name: "Review changes" })).toBeInTheDocument();
      expect(dialog).toHaveTextContent("mention /refine-task");
      if (system) {
        expect(dialog).toHaveTextContent("System skills cannot be deleted.");
        expect(within(dialog).queryByRole("button", { name: "Delete anyway" })).not.toBeInTheDocument();
      } else {
        expect(within(dialog).getByRole("button", { name: "Delete anyway" })).toBeEnabled();
      }
      await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
      expect(screen.getByTestId("agent-resource-skill-command")).toHaveValue("/refine-task");
    }, 15000);
  }
  it("does not allow deletion when usage cannot be checked", async () => {
    const user = userEvent.setup();
    renderEditor(false, true);
    await user.click(await screen.findByRole("button", { name: "Delete" }));
    const dialog = await screen.findByRole("alertdialog");
    expect(
      await within(dialog).findByText("Could not check automation usage. Try again before deleting this skill."),
    ).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Delete" })).toBeDisabled();
    expect(within(dialog).getByRole("button", { name: "Retry" })).toBeEnabled();
  }, 15000);
  it("keeps normal deletion available when no prompt mentions the skill", async () => {
    const user = userEvent.setup();
    renderEditor(false, false, false);
    await user.click(await screen.findByRole("button", { name: "Delete" }));
    const dialog = await screen.findByRole("alertdialog");
    const confirm = within(dialog).getByRole("button", { name: "Delete" });
    await waitFor(() => expect(confirm).toBeEnabled());
    expect(within(dialog).queryByRole("link", { name: "Review changes" })).not.toBeInTheDocument();
  }, 15000);
});
