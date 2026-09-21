import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import githubIcon from "@/assets/icons/integrations/github.svg";
import productiveIcon from "@/assets/icons/integrations/productive.svg";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { BacklogCreatePopover } from "./BacklogCreatePopover";
import type { BacklogIntakeItem, BacklogIntakeSource } from "./backlogIntakeItems";

const GITHUB_SOURCE: BacklogIntakeSource = {
  intakeId: "intake-github",
  name: "GitHub issues",
  iconSrc: githubIcon,
  iconAlt: "GitHub",
};

const PRODUCTIVE_SOURCE: BacklogIntakeSource = {
  intakeId: "intake-productive",
  name: "Productive.io tasks",
  iconSrc: productiveIcon,
  iconAlt: "Productive.io",
};

const GITHUB_ITEMS: BacklogIntakeItem[] = [
  {
    id: "gh-1",
    intakeId: "intake-github",
    key: "#12",
    title: "Handle duplicate refunds",
    body: "Retrying a refund posts twice.",
  },
  {
    id: "gh-2",
    intakeId: "intake-github",
    key: "#18",
    title: "Fix checkout timeout",
    body: "",
  },
];

const PRODUCTIVE_ITEMS: BacklogIntakeItem[] = [
  {
    id: "pr-1",
    intakeId: "intake-productive",
    key: "T-41",
    title: "Sync billing status",
    body: "",
  },
];

function CreateMenuStory({ sources, catalog }: { sources: BacklogIntakeSource[]; catalog: BacklogIntakeItem[] }) {
  const [query, setQuery] = useState("");
  const [focusedIntakeId, setFocusedIntakeId] = useState<string | null>(null);
  const items = catalog.filter((item) => item.intakeId === focusedIntakeId);

  return (
    <BacklogCreatePopover
      canAdd
      sources={sources}
      items={items}
      query={query}
      focusedIntakeId={focusedIntakeId}
      onQueryChange={setQuery}
      onFocusedIntakeChange={setFocusedIntakeId}
      onCreateManually={() => undefined}
      onImportItem={() => undefined}
    />
  );
}

const meta = {
  title: "Factories/Components/BacklogCreatePopover",
  component: BacklogCreatePopover,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="relative min-h-[360px] bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof BacklogCreatePopover>;

export default meta;

type Story = StoryObj<typeof meta>;

export const OneSource: Story = {
  name: "One source",
  render: () => <CreateMenuStory sources={[GITHUB_SOURCE]} catalog={GITHUB_ITEMS} />,
};

export const TwoSources: Story = {
  name: "Two sources",
  render: () => (
    <CreateMenuStory sources={[GITHUB_SOURCE, PRODUCTIVE_SOURCE]} catalog={[...GITHUB_ITEMS, ...PRODUCTIVE_ITEMS]} />
  ),
};
