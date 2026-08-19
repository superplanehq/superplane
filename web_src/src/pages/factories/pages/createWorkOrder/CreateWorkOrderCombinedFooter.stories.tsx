import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { expect, userEvent, within } from "storybook/test";
import { useState } from "react";

import type { SuperplaneUsersUser } from "@/api-client";
import { organizationKeys } from "@/hooks/useOrganizationData";

import { REFUND_FACTORY_LINES } from "../../__fixtures__/factoryPageResponses";
import { CreateWorkOrderCombinedFooter } from "./CreateWorkOrderCombinedFooter";

const ORGANIZATION_ID = "org-storybook";

const STORYBOOK_USERS: SuperplaneUsersUser[] = [
  { metadata: { id: "user-alex", email: "alex@superplane.dev" }, spec: { displayName: "Alex Reviewer" } },
  { metadata: { id: "user-priya", email: "priya@superplane.dev" }, spec: { displayName: "Priya Singh" } },
];

function createQueryClient() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
  queryClient.setQueryData(organizationKeys.users(ORGANIZATION_ID), STORYBOOK_USERS);
  return queryClient;
}

/**
 * Support stories only (issue #6791) — the reviewed surface is the Create
 * Work Order page story. This shows the redesigned footer states in
 * isolation: Owner + Save as draft + Send to line, with line choice moved
 * to send time.
 */
function FooterHarness({ lines = REFUND_FACTORY_LINES }: { lines?: typeof REFUND_FACTORY_LINES }) {
  const [queryClient] = useState(createQueryClient);
  const [assigneeIds, setAssigneeIds] = useState<string[]>([]);
  const [isSavingDraft, setIsSavingDraft] = useState(false);
  const [isSendingToLine, setIsSendingToLine] = useState(false);
  const [sentLine, setSentLine] = useState<string | null>(null);

  return (
    <QueryClientProvider client={queryClient}>
      <div className="mx-auto w-full max-w-2xl overflow-hidden rounded-xl border border-border bg-background shadow-sm">
        <div className="px-5 py-8 text-center text-[13px] text-muted-foreground">
          {sentLine ? `Sent to ${sentLine}` : "New work order composer"}
        </div>
        <CreateWorkOrderCombinedFooter
          organizationId={ORGANIZATION_ID}
          assigneeIds={assigneeIds}
          lines={lines}
          isSaving={isSavingDraft || isSendingToLine}
          canDispatch
          canSaveDraft
          isSavingDraft={isSavingDraft}
          isSendingToLine={isSendingToLine}
          onAssigneeChange={setAssigneeIds}
          onSaveDraft={() => {
            setIsSavingDraft(true);
            window.setTimeout(() => setIsSavingDraft(false), 400);
          }}
          onSendToLine={(lineName) => {
            setIsSendingToLine(true);
            window.setTimeout(() => {
              setIsSendingToLine(false);
              setSentLine(lineName);
            }, 400);
          }}
        />
      </div>
    </QueryClientProvider>
  );
}

const meta = {
  title: "Factories/Components/Create Work Order Combined Footer",
  parameters: { layout: "padded" },
} satisfies Meta<typeof FooterHarness>;

export default meta;

type Story = StoryObj<typeof meta>;

export const LinesAvailable: Story = {
  name: "Lines available",
  render: () => <FooterHarness />,
};

export const SendToLinePopoverOpen: Story = {
  name: "Send to line popover open",
  render: () => <FooterHarness />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByTestId("work-order-create-send-to-line"));
    await expect(await canvas.findByTestId("work-order-line-picker-panel")).toBeInTheDocument();
  },
};

export const NoLines: Story = {
  name: "No lines",
  render: () => <FooterHarness lines={[]} />,
};
