import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoryListPage } from "./FactoryListPage";
import { FactoriesHarness } from "./__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, emptyFactoriesFixture } from "./__fixtures__/factoryPageResponses";

/**
 * Mounts the real Factories list route against an in-process fixture backend.
 * Use **Populated** for the baseline list with a Refunds Factory + empty Payments
 * Factory, and **Empty** for the first-run empty state.
 *
 * Networking is faked via `window.fetch` (same rationale as HomePage/AppPage) —
 * see `OrgWorkspaceHarness` for the shared shell.
 */
const meta = {
  title: "Pages/Factories/List",
  component: FactoryListPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof FactoryListPage>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Populated list view: two factories, "Create factory" CTA available. */
export const Populated: Story = {
  render: () => <FactoriesHarness factoriesFixture={defaultFactoriesFixture} />,
};

/** First-run experience: no factories yet — dashed empty state with primary CTA. */
export const Empty: Story = {
  render: () => <FactoriesHarness factoriesFixture={emptyFactoriesFixture} />,
};
