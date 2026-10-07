import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";

import type { AdminIntakeEntry } from "./intakeCatalogModel";
import { MaturityStepper } from "./MaturityStepper";

const meta = {
  title: "Admin/Intakes/MaturityStepper",
  component: MaturityStepper,
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-screen bg-slate-100 p-6 dark:bg-gray-950">
        <div className="max-w-xl">
          <Story />
        </div>
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof MaturityStepper>;

export default meta;
type Story = StoryObj<typeof meta>;

const ENTRY: AdminIntakeEntry = {
  key: "jira-issues",
  name: "Jira issues",
  category: "issue_tracking",
  status: "beta",
  status_note: "",
  implemented: true,
  deletable: false,
  created_at: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
  updated_at: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
  updated_by_name: "Ada Lovelace",
};

export const Implemented: Story = {
  render: () => {
    const [entry, setEntry] = useState<AdminIntakeEntry>(ENTRY);
    return (
      <MaturityStepper
        entry={entry}
        pending={false}
        onChangeStatus={(status) => {
          // eslint-disable-next-line no-console
          console.log("onChangeStatus", status);
          setEntry((prev) => ({ ...prev, status }));
        }}
      />
    );
  },
};

export const NotImplemented: Story = {
  render: () => (
    <MaturityStepper
      entry={{
        ...ENTRY,
        key: "acme-tracker",
        name: "Acme tracker",
        status: "planned",
        implemented: false,
        deletable: true,
      }}
      pending={false}
      onChangeStatus={(status) => {
        // eslint-disable-next-line no-console
        console.log("onChangeStatus", status);
      }}
    />
  ),
};

export const Deprecated: Story = {
  render: () => (
    <MaturityStepper
      entry={{ ...ENTRY, status: "deprecated" }}
      pending={false}
      onChangeStatus={(status) => {
        // eslint-disable-next-line no-console
        console.log("onChangeStatus", status);
      }}
    />
  ),
};
