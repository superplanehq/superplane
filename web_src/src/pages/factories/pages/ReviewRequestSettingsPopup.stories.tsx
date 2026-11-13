import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { reviewRequestCanvas } from "../__fixtures__/reviewRequestFixture";
import { applyNodeConfigurations } from "./nodeConfigurationCanvas";
import { ReviewRequestSettingsPopup } from "./ReviewRequestSettingsPopup";
import {
  factoryHistoryHandlers,
  factoryHistoryStoryFactory,
  factoryHistoryStoryOrganization,
} from "./automationRunHistory/factoryRunHistoryFixtures";

function ReviewRequestPreview({
  empty = false,
  loading = false,
  error = false,
  saving = false,
  failSave = false,
}: {
  empty?: boolean;
  loading?: boolean;
  error?: boolean;
  saving?: boolean;
  failSave?: boolean;
}) {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const [canvas, setCanvas] = useState(() => {
    const canvas = reviewRequestCanvas();
    if (empty) canvas.spec!.nodes![1].configuration!.fallbackReviewer = "";
    canvas.spec!.nodes![1].configuration!.additionalInstructions = empty
      ? ""
      : "Prefer the security owner for authentication changes. Explain the ownership evidence for each reviewer.";
    return canvas;
  });
  const [unavailable, setUnavailable] = useState(loading || error);
  const [shouldFailSave, setShouldFailSave] = useState(failSave);
  return (
    <QueryClientProvider client={client}>
      <ComponentStoryShell initialPath="/?automationTab=general" className="h-screen bg-background">
        <ReviewRequestSettingsPopup
          organizationId={factoryHistoryStoryOrganization}
          factoryId={factoryHistoryStoryFactory}
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
            if (saving) await new Promise<void>(() => {});
            if (shouldFailSave) {
              setShouldFailSave(false);
              throw new globalThis.Error("Could not save Review request. Try again.");
            }
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
export const Saving: Story = {
  args: { saving: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("review-request-settings-save"));
    await expect(canvas.getByRole("button", { name: "Saving" })).toBeDisabled();
    await expect(canvas.getByRole("textbox", { name: "Additional instructions" })).toBeDisabled();
  },
};
export const SaveError: Story = {
  args: { failSave: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(await canvas.findByTestId("review-request-settings-save"));
    await expect(await canvas.findByRole("alert")).toHaveTextContent("Could not save Review request. Try again.");
    await expect(canvas.getByTestId("review-request-settings-save")).toBeEnabled();
  },
};
