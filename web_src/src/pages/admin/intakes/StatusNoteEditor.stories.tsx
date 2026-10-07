import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";

import type { AdminIntakeEntry } from "./intakeCatalogModel";
import { StatusNoteEditor } from "./IntakeDetailParts";

const meta = {
  title: "Admin/Intakes/StatusNoteEditor",
  component: StatusNoteEditor,
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-screen bg-slate-100 p-6 dark:bg-gray-950">
        <div className="max-w-xl rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
          <Story />
        </div>
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof StatusNoteEditor>;

export default meta;
type Story = StoryObj<typeof meta>;

const ENTRY: AdminIntakeEntry = {
  key: "datadog",
  name: "Datadog errors",
  category: "error_tracking",
  status: "beta",
  status_note: "Outside testers must confirm that it works.",
  implemented: true,
  deletable: false,
  created_at: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
  updated_at: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
  updated_by_name: "Ada Lovelace",
};

export const Default: Story = {
  render: () => {
    const [note, setNote] = useState(ENTRY.status_note);
    return (
      <StatusNoteEditor
        entry={{ ...ENTRY, status_note: note }}
        saving={false}
        savedAt={null}
        onSave={(next) => {
          // eslint-disable-next-line no-console
          console.log("onSave", next);
          setNote(next);
        }}
      />
    );
  },
};

export const Saving: Story = {
  render: () => (
    <StatusNoteEditor
      entry={ENTRY}
      saving
      savedAt={Date.now() - 15_000}
      onSave={(next) => {
        // eslint-disable-next-line no-console
        console.log("onSave", next);
      }}
    />
  ),
};
