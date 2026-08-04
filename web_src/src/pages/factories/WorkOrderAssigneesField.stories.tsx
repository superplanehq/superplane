import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  FACTORIES_ORGANIZATION_ID,
  REVIEWER_USER,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
} from "./__fixtures__/factoryPageResponses";
import { WorkOrderAssigneesField } from "./WorkOrderAssigneesField";

/**
 * Sidebar assignees panel: heading + edit affordance + list. The popover is
 * gated to view-only in these stories to avoid the picker's live fetch;
 * the harnessed full-page stories cover the interactive path.
 */
const meta = {
  title: "Factories/WorkOrderAssigneesField",
  component: WorkOrderAssigneesField,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[220px] w-[320px] bg-white p-6 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderAssigneesField>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Multiple assignees, editable. */
export const WithAssignees: Story = {
  name: "With Assignees",
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    assigneeIds: [STORYBOOK_ME_USER_ID, REVIEWER_USER.id],
    assigneeNames: [STORYBOOK_ME_USER_NAME, REVIEWER_USER.name],
    canEdit: true,
    isSaving: false,
    onSave: async (ids) => {
      console.log("save assignees", ids);
    },
  },
};

/** Empty state — "No one assigned" copy. */
export const Empty: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    assigneeIds: [],
    assigneeNames: [],
    canEdit: true,
    isSaving: false,
    onSave: async () => undefined,
  },
};

/** Read-only viewer — edit button disabled with tooltip. */
export const ReadOnly: Story = {
  name: "Read Only",
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    assigneeIds: [STORYBOOK_ME_USER_ID],
    assigneeNames: [STORYBOOK_ME_USER_NAME],
    canEdit: false,
    isSaving: false,
    onSave: async () => undefined,
  },
};
