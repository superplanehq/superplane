import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { http, HttpResponse } from "msw";
import { Button } from "@/components/ui/button";
import { MobileDraftStartModelSelect } from "../../mobile/MobileDraftStartModelSelect";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import type { DraftStartSelection } from "./draftStartModel";

const modelName = "openai/gpt-6.1-sol";
const model = `hosted::openrouter::${modelName}`;
const meta = {
  title: "Factories/Start defaults",
  component: DraftStartModelSelect,
  parameters: {
    msw: {
      handlers: [
        http.get("*/line-runner-models", () =>
          HttpResponse.json({
            models: [
              { id: model, name: modelName },
              { id: `hosted::openai::gpt-6.1-sol`, name: modelName },
              { id: `byok::openrouter::${modelName}`, name: modelName },
              { id: modelName, name: modelName },
              { id: "long-model", name: "Anthropic Claude Opus 4.6 with extended context" },
            ],
          }),
        ),
      ],
    },
  },
} satisfies Meta<typeof DraftStartModelSelect>;
export default meta;
type Story = StoryObj<typeof meta>;

function Preview({
  mobile = false,
  loading = false,
  failed = false,
  initialSelection = {},
  appearance = "labeled",
}: {
  mobile?: boolean;
  loading?: boolean;
  failed?: boolean;
  initialSelection?: DraftStartSelection;
  appearance?: "labeled" | "ghost" | "icon";
}) {
  const [selection, onChange] = useState<DraftStartSelection>(initialSelection);
  const [ready, setReady] = useState(!loading && !failed);
  const defaults = {
    values: ready ? { model, thinkingLevel: "high" } : undefined,
    isLoading: !ready && !failed,
    isError: !ready && failed,
    retry: () => setReady(true),
  };
  const props = {
    ...selection,
    onChange,
    defaults,
    organizationId: "org-preview",
    factoryId: "factory-preview",
    lineName: "ship",
  };
  return (
    <div className="p-6">
      <div className="flex max-w-full items-center rounded-md border">
        <Button size="sm">Start</Button>
        {mobile ? (
          <MobileDraftStartModelSelect {...props} />
        ) : (
          <DraftStartModelSelect {...props} appearance={appearance} />
        )}
      </div>
    </div>
  );
}

const decorators: Story["decorators"] = [
  (Story) => (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <Story />
    </QueryClientProvider>
  ),
];
export const Desktop: Story = { decorators, render: () => <Preview /> };
export const Mobile: Story = { decorators, render: () => <Preview mobile /> };
export const Loading: Story = { decorators, render: () => <Preview loading /> };
export const Retry: Story = { decorators, render: () => <Preview failed /> };
export const MobileRetry: Story = { decorators, render: () => <Preview mobile failed /> };
export const Overrides: Story = {
  decorators,
  render: () => <Preview initialSelection={{ model: modelName, thinkingLevel: "low" }} />,
};
export const MobileOverrides: Story = {
  decorators,
  render: () => <Preview mobile initialSelection={{ thinkingLevel: "high" }} />,
};
export const RefineStrip: Story = { decorators, render: () => <Preview appearance="ghost" /> };
export const Icon: Story = { decorators, render: () => <Preview appearance="icon" /> };
