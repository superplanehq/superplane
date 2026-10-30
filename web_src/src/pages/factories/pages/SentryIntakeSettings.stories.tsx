import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { usersHandler } from "./IntakeDefaultOwnerField.stories";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { DEFAULT_SENTRY_INTAKE_SETTINGS } from "./intakeSourceSettingsModel";
import {
  factoryHistoryHandlers,
  factoryHistoryStoryCanvasId,
  factoryHistoryStoryFactory,
  factoryHistoryStoryOrganization,
} from "./automationRunHistory/factoryRunHistoryFixtures";

function SentryIntakePreview() {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const [settings, setSettings] = useState({
    ...DEFAULT_SENTRY_INTAKE_SETTINGS,
    sentryProjectIds: ["payments"],
  });
  return (
    <QueryClientProvider client={client}>
      <ComponentStoryShell className="h-screen bg-background">
        <IntakeSourceSettingsPopup
          sourceId="sentry-exceptions"
          organizationId={factoryHistoryStoryOrganization}
          factoryId={factoryHistoryStoryFactory}
          canvasId={factoryHistoryStoryCanvasId("sentry-exceptions")}
          settings={settings}
          initialTab="general"
          onSave={(next) => {
            console.log("Save Sentry intake settings", next);
            setSettings(next);
          }}
          onDelete={() => console.log("Delete Sentry intake")}
          onClose={() => console.log("Close Sentry intake")}
        />
      </ComponentStoryShell>
    </QueryClientProvider>
  );
}

const meta = {
  title: "Factories/Pages/Sentry intake settings",
  component: SentryIntakePreview,
  decorators: [withFactoriesTheme],
  parameters: {
    layout: "fullscreen",
    msw: { handlers: [usersHandler("ready"), ...factoryHistoryHandlers("sentry-exceptions")] },
  },
} satisfies Meta<typeof SentryIntakePreview>;
export default meta;
type Story = StoryObj<typeof meta>;

export const General: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body);
    await userEvent.click(await canvas.findByRole("tab", { name: "General" }));
    await expect(await canvas.findByRole("button", { name: "Default task owner" })).toBeVisible();
  },
};
