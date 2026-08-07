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
 * Sidebar assignees panel: heading + edit affordance + list. These stories
 * showcase the panel's *rendered* states (populated / empty / read-only);
 * the assignees popover fetches from `/api/v1/users` when opened, so the
 * interactive edit path is covered by the harnessed page stories under
 * `Pages/Factories/WorkOrderDetail`. Edit is disabled here to keep every
 * story self-contained.
 */
const meta = {
  title: "Factories/Components/WorkOrderAssigneesField",
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

/** Multiple assignees rendered as chips. */
export const WithAssignees: Story = {
  name: "With Assignees",
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    assigneeIds: [STORYBOOK_ME_USER_ID, REVIEWER_USER.id],
    assigneeNames: [STORYBOOK_ME_USER_NAME, REVIEWER_USER.name],
    canEdit: false,
    isSaving: false,
    onSave: async () => undefined,
  },
};

/** Empty state — "No one assigned" copy. */
export const Empty: Story = {
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    assigneeIds: [],
    assigneeNames: [],
    canEdit: false,
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
