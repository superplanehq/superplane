import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { reviewRequestCanvas } from "../__fixtures__/reviewRequestFixture";
import { applyNodeConfigurations } from "./nodeConfigurationCanvas";
import { ReviewRequestSettingsPopup } from "./ReviewRequestSettingsPopup";
import { factoryHistoryHandlers } from "./automationRunHistory/factoryRunHistoryFixtures";

function ReviewRequestPreview({
  empty = false,
  loading = false,
  error = false,
}: {
  empty?: boolean;
  loading?: boolean;
  error?: boolean;
}) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const [canvas, setCanvas] = useState(() => {
    const canvas = reviewRequestCanvas();
    if (empty) canvas.spec!.nodes![1].configuration!.fallbackReviewer = "";
    return canvas;
  });
  const [unavailable, setUnavailable] = useState(loading || error);
  return (
    <QueryClientProvider client={client}>
      <ComponentStoryShell initialPath="/?automationTab=general" className="h-screen bg-background">
        <ReviewRequestSettingsPopup
          organizationId="org-1"
          factoryId="factory-1"
          canvasId="review-request-app"
          graph={{ nodes: [], edges: [], specNodes: canvas.spec?.nodes }}
          reviewers={empty ? [] : ["saved-reviewer", "alex", "sam"]}
          reviewersLoading={loading && unavailable}
          reviewersError={error && unavailable}
          onRetryReviewers={() => {
            console.log("Retry GitHub reviewers");
            setUnavailable(false);
          }}
          onSaveNode={async (update) => {
            console.log("Save Review request", update);
            setCanvas((current) => applyNodeConfigurations(current, Array.isArray(update) ? update : [update]));
          }}
          onDelete={() => console.log("Delete Review request")}
          onClose={() => console.log("Close Review request")}
        />
      </ComponentStoryShell>
    </QueryClientProvider>
  );
}

const meta = {
  title: "Factories/Pages/Review request settings",
  component: ReviewRequestPreview,
  decorators: [withFactoriesTheme],
  parameters: { layout: "fullscreen", msw: { handlers: factoryHistoryHandlers("planning") } },
} satisfies Meta<typeof ReviewRequestPreview>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Configured: Story = {};
export const Empty: Story = { args: { empty: true } };
export const Loading: Story = { args: { loading: true } };
export const Error: Story = { args: { error: true } };
