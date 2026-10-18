import type { Meta, StoryObj } from "@storybook/react-vite";
import { RestoreSystemSkillDialog } from "./RestoreSystemSkillDialog";

const meta = {
  title: "Factory/Agent/Restore system skill",
  component: RestoreSystemSkillDialog,
  args: {
    open: true,
    isRestoring: false,
    onClose: () => console.log("Cancel restore"),
    onConfirm: async () => {
      console.log("Restore skill");
    },
  },
} satisfies Meta<typeof RestoreSystemSkillDialog>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Confirmation: Story = {};
export const Restoring: Story = { args: { isRestoring: true } };
