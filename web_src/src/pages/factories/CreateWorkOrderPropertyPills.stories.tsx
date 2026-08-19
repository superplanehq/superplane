import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";

import type { SuperplaneUsersUser } from "@/api-client";
import { organizationKeys } from "@/hooks/useOrganizationData";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  FACTORIES_ORGANIZATION_ID,
  ORGANIZATION_USERS,
  REFUND_FACTORY_LINES,
  REVIEWER_USER,
  STORYBOOK_ME_USER_ID,
} from "./__fixtures__/factoryPageResponses";
import { CreateWorkOrderPropertyPills } from "./CreateWorkOrderPropertyPills";

const SEEDED_USERS: SuperplaneUsersUser[] = ORGANIZATION_USERS.map((user) => ({
  metadata: { id: user.id, email: user.email },
  spec: { displayName: user.name },
}));

function createSeededQueryClient() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity, retry: false } } });
  queryClient.setQueryData(organizationKeys.users(FACTORIES_ORGANIZATION_ID), SEEDED_USERS);
  return queryClient;
}

/**
 * Footer property pills from the New Work Order dialog. Isolated here so the
 * Owner default-current-user, changed, and cleared states are easy to
 * compare side by side. The default is also visible live in
 * `Factories/Pages/Create Work Order` — this story is a supplement, not the
 * only place to review it.
 */
const meta = {
  title: "Factories/Components/CreateWorkOrderPropertyPills",
  component: CreateWorkOrderPropertyPills,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <QueryClientProvider client={createSeededQueryClient()}>
        <ComponentStoryShell className="flex min-h-[160px] items-center justify-center bg-gray-50 p-6 dark:bg-gray-950">
          <Story />
        </ComponentStoryShell>
      </QueryClientProvider>
    ),
  ],
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    assigneeIds: [],
    lines: REFUND_FACTORY_LINES,
    selectedLineName: "",
    isSaving: false,
    onAssigneeChange: () => {},
    onLineSelect: () => {},
  },
} satisfies Meta<typeof CreateWorkOrderPropertyPills>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Mirrors the composer: Owner starts pre-filled and the picker keeps working on top of it. */
function InteractiveOwnerPill({ initialAssigneeIds }: { initialAssigneeIds: string[] }) {
  const [assigneeIds, setAssigneeIds] = useState(initialAssigneeIds);
  const [selectedLineName, setSelectedLineName] = useState("");

  return (
    <CreateWorkOrderPropertyPills
      organizationId={FACTORIES_ORGANIZATION_ID}
      assigneeIds={assigneeIds}
      lines={REFUND_FACTORY_LINES}
      selectedLineName={selectedLineName}
      isSaving={false}
      onAssigneeChange={setAssigneeIds}
      onLineSelect={setSelectedLineName}
    />
  );
}

/** New Work Order opens with Owner already showing the creator — no extra click to assign yourself. Open the pill to see the current user pre-selected in the picker. */
export const OwnerDefaultedToCurrentUser: Story = {
  name: "Owner defaulted to current user",
  render: () => <InteractiveOwnerPill initialAssigneeIds={[STORYBOOK_ME_USER_ID]} />,
};

/** The default is only a starting point — the operator can reassign to a teammate instead. */
export const OwnerChangedToSomeoneElse: Story = {
  name: "Owner changed to someone else",
  args: {
    assigneeIds: [REVIEWER_USER.id],
  },
};

/** Clearing every assignee falls back to the empty "Owner" label — the default never forces ownership. */
export const OwnerCleared: Story = {
  name: "Owner cleared (empty)",
  args: {
    assigneeIds: [],
  },
};
