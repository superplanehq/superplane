import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { expect, userEvent, within } from "storybook/test";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { IntakeSourceSettingsPopup } from "./IntakeSourceSettingsPopup";
import { DEFAULT_GITHUB_INTAKE_SETTINGS } from "./intakeSourceSettingsModel";
import {
  factoryHistoryHandlers,
  factoryHistoryStoryCanvasId,
  factoryHistoryStoryFactory,
  factoryHistoryStoryOrganization,
} from "./automationRunHistory/factoryRunHistoryFixtures";

function GitHubIntakePreview({
  initiallyPaused = false,
  failPause = false,
  allIssues = false,
  labelsLoading = false,
}: {
  initiallyPaused?: boolean;
  failPause?: boolean;
  allIssues?: boolean;
  labelsLoading?: boolean;
}) {
  const [client] = useState(() => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
    client.setQueryData(["organization", "users", factoryHistoryStoryOrganization], {
      users: [
        { metadata: { id: "alex" }, spec: { displayName: "Alex Morgan", email: "alex@example.com" } },
        { metadata: { id: "sam" }, spec: { displayName: "Sam Rivera", email: "sam@example.com" } },
      ],
    });
    return client;
  });
  const [settings, setSettings] = useState({
    ...DEFAULT_GITHUB_INTAKE_SETTINGS,
    filterByLabel: !allIssues,
    labels: allIssues ? [] : ["superplane"],
  });
  const [paused, setPaused] = useState(initiallyPaused);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string>();
  const failNext = useRef(failPause);
  return (
    <QueryClientProvider client={client}>
      <ComponentStoryShell className="h-screen bg-background">
        <IntakeSourceSettingsPopup
          sourceId="github-issues"
          organizationId={factoryHistoryStoryOrganization}
          factoryId={factoryHistoryStoryFactory}
          canvasId={factoryHistoryStoryCanvasId("github-issues")}
          repository="superplanehq/superplane"
          settings={settings}
          labelOptions={["superplane", "bug", "needs-triage"]}
          labelOptionsLoading={labelsLoading}
          paused={paused}
          savePending={pending}
          saveError={error}
          onPausedChange={async (next) => {
            setError(undefined);
            setPending(true);
            try {
              await new Promise((resolve) => setTimeout(resolve, 300));
              if (failNext.current) {
                failNext.current = false;
                throw new Error("SuperPlane could not update the intake. Try again.");
              }
              console.log("Set GitHub intake paused", next);
              setPaused(next);
            } catch (cause) {
              setError(cause instanceof Error ? cause.message : "Could not update intake.");
              throw cause;
            } finally {
              setPending(false);
            }
          }}
          onSave={(next) => {
            console.log("Save GitHub intake settings", next);
            setSettings(next);
          }}
          onDelete={() => console.log("Delete GitHub intake")}
          onClose={() => console.log("Close GitHub intake")}
        />
      </ComponentStoryShell>
    </QueryClientProvider>
  );
}

const meta = {
  title: "Factories/Pages/GitHub intake settings",
  component: GitHubIntakePreview,
  decorators: [withFactoriesTheme],
  parameters: { layout: "fullscreen", msw: { handlers: factoryHistoryHandlers("github-issues") } },
} satisfies Meta<typeof GitHubIntakePreview>;
export default meta;
type Story = StoryObj<typeof meta>;

export const Configuration: Story = {};
export const AllIssues: Story = { args: { allIssues: true } };
export const LabelsLoading: Story = { args: { labelsLoading: true } };
export const Paused: Story = { args: { initiallyPaused: true } };
export const PauseError: Story = {
  args: { failPause: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body);
    await userEvent.click(await canvas.findByTestId("intake-source-settings-automatic-intake"));
    await expect(await canvas.findByRole("alert")).toHaveTextContent("Try again.");
  },
};
