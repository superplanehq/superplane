import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "bun:test";

import { client } from "@/api-client/client.gen";
import { FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { INLINE_SKILL, UI_UX_PRO_MAX_SKILL } from "../../__fixtures__/agentResourceFixtures";

const DISABLED_SKILL = {
  ...INLINE_SKILL,
  id: "resource-disabled-skill",
  name: "disabled-skill",
  enabled: false,
  markdown: "---\nname: disabled-skill\ntitle: Disabled skill\ndescription: Off for workspace.\n---\n\nBody.",
};
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

const agentPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/agent`;
const skillsEditorPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/workspace/skills`;
const mcpAndSkills = [FEATURE_WORKSPACE_MCP, FEATURE_WORKSPACE_SKILLS];

describe("FactorySettingsSkillsPage", () => {
  beforeAll(() => {
    client.setConfig({ baseUrl: "http://localhost" });
    Element.prototype.scrollIntoView ??= vi.fn();
  });

  it("shows the skills empty state", async () => {
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("agent-resources-skills-empty", {}, { timeout: 8000 })).toHaveTextContent(
      "No skills yet",
    );
    expect(screen.getByTestId("agent-resources-add-skill")).toBeEnabled();
    expect(screen.getByTestId("agent-resources-add-skill")).toHaveTextContent("Add skill");
  }, 10000);

  it("opens the full-page skill editor", async () => {
    render(
      <FactoriesHarness
        pathSuffix={`${skillsEditorPath}/new`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("factory-settings-skill-editor", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-skill-name")).toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-skill-command")).toHaveValue("/");
    expect(screen.getByTestId("agent-resource-skill-markdown")).toBeInTheDocument();
  }, 10000);

  it("recommends a slash command from the skill name", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${skillsEditorPath}/new`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.type(await screen.findByTestId("agent-resource-skill-name", {}, { timeout: 8000 }), "review-copy");
    expect(screen.getByTestId("agent-resource-skill-command")).toHaveValue("/review-copy");
  }, 10000);

  it("strips spaces and punctuation from the recommended command", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${skillsEditorPath}/new`}
        factoriesFixture={defaultFactoriesFixture}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.type(await screen.findByTestId("agent-resource-skill-name", {}, { timeout: 8000 }), "Oy Pirate!");
    expect(screen.getByTestId("agent-resource-skill-command")).toHaveValue("/oypirate");
    expect(screen.getByTestId("agent-resource-skill-name")).toHaveValue("Oy Pirate!");
    expect(screen.queryByText("The SKILL.md name does not match this skill name.")).not.toBeInTheDocument();
  }, 10000);

  it("lists an inline skill", async () => {
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [INLINE_SKILL] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("agent-resources-skills-list", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByText("Review copy")).toBeInTheDocument();
    expect(screen.queryByText("Review UI copy.")).not.toBeInTheDocument();
    expect(screen.getByTestId(`agent-resource-configure-${INLINE_SKILL.id}`)).toBeInTheDocument();
  }, 10000);

  it("opens edit when the skill name is clicked", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [INLINE_SKILL] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    await user.click(await screen.findByTestId(`agent-resource-edit-${INLINE_SKILL.id}`, {}, { timeout: 8000 }));
    expect(await screen.findByTestId("factory-settings-skill-editor", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByTestId("agent-resource-skill-name")).toHaveValue("Review copy");
  }, 10000);

  it("re-enables a disabled skill from the editor", async () => {
    const user = userEvent.setup();
    render(
      <FactoriesHarness
        pathSuffix={`${skillsEditorPath}/${DISABLED_SKILL.id}`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [DISABLED_SKILL] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    const enabledSwitch = await screen.findByTestId("skill-editor-enabled", {}, { timeout: 8000 });
    expect(enabledSwitch).not.toBeChecked();
    await user.click(enabledSwitch);
    expect(enabledSwitch).toBeChecked();
  }, 10000);

  it("lists a GitHub skill package", async () => {
    render(
      <FactoriesHarness
        pathSuffix={agentPath}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          agentResourcesByFactoryId: { [PRIMARY_FACTORY_ID]: [UI_UX_PRO_MAX_SKILL] },
        }}
        experimentalFeatures={mcpAndSkills}
      />,
    );

    expect(await screen.findByTestId("agent-resources-skills-list", {}, { timeout: 8000 })).toBeInTheDocument();
    expect(screen.getByText("ui-ux-pro-max")).toBeInTheDocument();
    expect(screen.queryByText("nextlevelbuilder/ui-ux-pro-max-skill@v1.2.0")).not.toBeInTheDocument();
  }, 10000);
});
