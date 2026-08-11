import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_ID } from "../__fixtures__/factoryPageResponses";
import { WikiWireframe } from "./wiki/WikiWireframe";
import { WIKI_DOCUMENTS_DEFAULT, WIKI_DOCUMENTS_REFRESHED } from "./wiki/wikiMocks";

/**
 * Storybook-only wiki wireframe inside FactoriesLayout.
 * The app `/wiki` route still renders Coming Soon.
 */
const meta = {
  title: "Factories/Pages/Wiki",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

const wikiPath = `workspaces/${PRIMARY_FACTORY_ID}/wiki`;

function DefaultWikiWireframe() {
  return <WikiWireframe initialDocuments={WIKI_DOCUMENTS_DEFAULT} refreshedDocuments={WIKI_DOCUMENTS_REFRESHED} />;
}

function EmptyWikiWireframe() {
  return <WikiWireframe initialDocuments={[]} refreshedDocuments={WIKI_DOCUMENTS_REFRESHED} />;
}

function EditingWikiWireframe() {
  return (
    <WikiWireframe
      initialDocuments={WIKI_DOCUMENTS_DEFAULT}
      refreshedDocuments={WIKI_DOCUMENTS_REFRESHED}
      startEditing
    />
  );
}

export const Default: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={wikiPath}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={{ wiki: DefaultWikiWireframe }}
    />
  ),
};

export const Empty: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={wikiPath}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={{ wiki: EmptyWikiWireframe }}
    />
  ),
};

export const Editing: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={wikiPath}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={{ wiki: EditingWikiWireframe }}
    />
  ),
};
