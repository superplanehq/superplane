import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { FactoriesHarness } from "./__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  FACTORIES_ORGANIZATION_ID,
  PRIMARY_FACTORY_ID,
} from "./__fixtures__/factoryPageResponses";
import { CreateWorkOrderForm } from "./CreateWorkOrderForm";

/**
 * The presentational form that composes the New Work Order page. Stories
 * wire up local state so the fields (including the assignees popover) are
 * fully interactive.
 */
const meta = {
  title: "Factories/CreateWorkOrderForm",
  component: CreateWorkOrderForm,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof CreateWorkOrderForm>;

export default meta;

type Story = StoryObj<typeof meta>;

const factoryHref = `/${FACTORIES_ORGANIZATION_ID}/factories/${PRIMARY_FACTORY_ID}`;

function InteractiveForm({
  initialTitle = "",
  initialDescription = "",
  isCreating = false,
}: {
  initialTitle?: string;
  initialDescription?: string;
  isCreating?: boolean;
}) {
  const [title, setTitle] = useState(initialTitle);
  const [description, setDescription] = useState(initialDescription);
  const [assigneeIds, setAssigneeIds] = useState<string[]>([]);
  const [titleError, setTitleError] = useState("");

  return (
    <ComponentStoryShell className="min-h-screen w-full bg-gray-50 py-8 dark:bg-gray-950">
      <CreateWorkOrderForm
        organizationId={FACTORIES_ORGANIZATION_ID}
        factoryHref={factoryHref}
        factoryName="Refunds Factory"
        title={title}
        description={description}
        assigneeIds={assigneeIds}
        titleError={titleError}
        selectedAssigneeLabels={assigneeIds}
        isCreating={isCreating}
        onTitleChange={setTitle}
        onDescriptionChange={setDescription}
        onAssigneeIdsChange={setAssigneeIds}
        onClearTitleError={() => setTitleError("")}
        onCreate={() => {
          if (!title.trim()) {
            setTitleError("Title is required");
            return;
          }
          console.log("create", { title, description, assigneeIds });
        }}
      />
    </ComponentStoryShell>
  );
}

/**
 * Interactive form under the shared harness (so the assignees popover can
 * fetch the seeded org roster).
 */
export const Default: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={`factories/${PRIMARY_FACTORY_ID}/orders/new`}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

/**
 * Isolated form (no harness) — pure component with local state. Assignees
 * popover will show "Loading members…" without a fixture backend; useful for
 * reviewing the form scaffolding alone.
 */
export const Isolated: Story = {
  render: () => <InteractiveForm />,
};

/** Prefilled form — represents a mid-editing snapshot. */
export const Prefilled: Story = {
  render: () => (
    <InteractiveForm
      initialTitle="Add refund reconciliation test"
      initialDescription="Cover the newly discovered duplicate refund case with a regression test running in CI."
    />
  ),
};

/** Saving state — the primary CTA shows the loading label. */
export const Saving: Story = {
  render: () => (
    <InteractiveForm
      initialTitle="Add refund reconciliation test"
      initialDescription="Cover the newly discovered duplicate refund case with a regression test running in CI."
      isCreating
    />
  ),
};
