import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import datadogIcon from "@/assets/icons/integrations/datadog.svg";
import dependabotIcon from "@/assets/icons/integrations/dependabot.svg";
import githubIcon from "@/assets/icons/integrations/github.svg";
import linearIcon from "@/assets/icons/integrations/linear.svg";
import productiveIcon from "@/assets/icons/integrations/productive.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { BacklogCreatePopover } from "./BacklogCreatePopover";
import type { BacklogIntakeItem, BacklogIntakeSource } from "./backlogIntakeItems";

const GITHUB_SOURCE: BacklogIntakeSource = {
  intakeId: "intake-github",
  name: "GitHub issues",
  iconSrc: githubIcon,
  iconAlt: "GitHub",
  tabLabel: "GitHub",
};

const PRODUCTIVE_SOURCE: BacklogIntakeSource = {
  intakeId: "intake-productive",
  name: "Productive tasks",
  iconSrc: productiveIcon,
  iconAlt: "Productive",
  tabLabel: "Productive",
};

const LINEAR_SOURCE: BacklogIntakeSource = {
  intakeId: "intake-linear",
  name: "Linear issues",
  iconSrc: linearIcon,
  iconAlt: "Linear",
  tabLabel: "Linear",
};

const DEPENDABOT_SOURCE: BacklogIntakeSource = {
  intakeId: "intake-dependabot",
  name: "Dependabot alerts",
  iconSrc: dependabotIcon,
  iconAlt: "Dependabot",
  tabLabel: "Dependabot",
};

const DATADOG_SOURCE: BacklogIntakeSource = {
  intakeId: "intake-datadog",
  name: "Datadog errors",
  iconSrc: datadogIcon,
  iconAlt: "Datadog",
  tabLabel: "Datadog",
};

const SENTRY_SOURCE: BacklogIntakeSource = {
  intakeId: "intake-sentry",
  name: "Sentry exceptions",
  iconSrc: sentryIcon,
  iconAlt: "Sentry",
  tabLabel: "Sentry",
};

const GITHUB_ITEMS: BacklogIntakeItem[] = [
  {
    id: "gh-1",
    intakeId: "intake-github",
    key: "#12",
    title: "Handle duplicate refunds",
    body: "Retrying a refund posts twice.",
    createdAt: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
  },
  {
    id: "gh-2",
    intakeId: "intake-github",
    key: "#18",
    title: "Fix checkout timeout",
    body: "",
    createdAt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: "gh-3",
    intakeId: "intake-github",
    key: "#21",
    title: "Retry failed webhook deliveries",
    body: "",
  },
  {
    id: "gh-4",
    intakeId: "intake-github",
    key: "#24",
    title: "Restore missing invoice PDFs",
    body: "",
  },
  {
    id: "gh-5",
    intakeId: "intake-github",
    key: "#27",
    title: "Stop double-charge on retry",
    body: "",
  },
  {
    id: "gh-6",
    intakeId: "intake-github",
    key: "#31",
    title: "Fix empty cart crash",
    body: "",
  },
  {
    id: "gh-7",
    intakeId: "intake-github",
    key: "#36",
    title: "Show expired coupon error",
    body: "",
  },
  {
    id: "gh-8",
    intakeId: "intake-github",
    key: "#40",
    title: "Keep search results on page error",
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
      onCreateManually={() => console.log("create work order manually")}
      onImportItem={(item) => console.log("import item", item)}
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

export const FourSources: Story = {
  name: "Four sources",
  render: () => (
    <CreateMenuStory
      sources={[LINEAR_SOURCE, DEPENDABOT_SOURCE, DATADOG_SOURCE, SENTRY_SOURCE]}
      catalog={GITHUB_ITEMS.map((item) => ({ ...item, intakeId: LINEAR_SOURCE.intakeId }))}
    />
  ),
};
