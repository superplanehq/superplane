import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { INLINE_SKILL, UI_UX_PRO_MAX_SKILL } from "../__fixtures__/agentResourceFixtures";
import { SkillPromptTextarea } from "./SkillPromptTextarea";

const meta = {
  title: "Factory/Agent/Skill prompt",
  component: SkillPromptTextarea,
  decorators: [
    (Story) => (
      <div className="mx-auto max-w-3xl p-6">
        <Story />
      </div>
    ),
  ],
  args: {
    value: "Use /review-copy to check the UI text.\n\nBuild the requested change.",
    skills: [INLINE_SKILL, UI_UX_PRO_MAX_SKILL],
    label: "Agent prompt",
    testId: "skill-prompt",
    onChange: () => {},
  },
  render: function Interactive(args) {
    const [value, setValue] = useState(args.value);
    return <SkillPromptTextarea {...args} value={value} onChange={setValue} />;
  },
} satisfies Meta<typeof SkillPromptTextarea>;
export default meta;
type Story = StoryObj<typeof meta>;
export const SkillsAvailable: Story = {};
export const NoAvailableSkills: Story = { args: { value: "Type / to find a skill.", skills: [] } };
