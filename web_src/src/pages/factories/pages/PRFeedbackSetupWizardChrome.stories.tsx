import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ChecksPRFeedbackSetupPreview } from "./ChecksPRFeedbackSetupPreview";
import { PRFeedbackSetupWizardShell } from "./PRFeedbackSetupWizardChrome";
import { PR_FEEDBACK_SETTINGS_COPY as copy } from "./prFeedbackSettingsCopy";
import { StatusCheckPicker } from "./StatusCheckPicker";

function ConfigurePreview({ empty = true }: { empty?: boolean }) {
  const [selectedNames, setSelectedNames] = useState(["lint", "test"]);
  const catalog = empty ? [] : ["lint", "test"].map((name) => ({ id: name, name, type: "status_check" }));
  return (
    <div className="h-svh">
      <PRFeedbackSetupWizardShell
        testId="configure-preview"
        preview={
          <ChecksPRFeedbackSetupPreview
            step="checks"
            catalog={catalog}
            selectedNames={selectedNames}
            catalogEmpty={empty}
            maximumAttempts={3}
            toolsAccess="suggested"
            suggestedNames={[]}
            runnerIntegrationIds={[]}
            connected={[]}
          />
        }
      >
        <header>
          <Button
            variant="ghost"
            className="mb-6 px-0 text-muted-foreground"
            onClick={() => console.info("Back to board")}
          >
            <ArrowLeft className="size-4" aria-hidden />
            Back to board
          </Button>
          <h1>{copy.wizardStepWhichChecks}</h1>
        </header>
        {empty ? (
          <div className="workspace-body-text space-y-2 text-muted-foreground">
            <p>{copy.wizardChecksEmpty}</p>
            <p>{copy.wizardChecksEmptyDetail}</p>
          </div>
        ) : (
          <StatusCheckPicker
            names={selectedNames}
            catalog={catalog}
            hideHeading
            onToggle={(name: string) =>
              setSelectedNames((names) =>
                names.includes(name) ? names.filter((item) => item !== name) : [...names, name],
              )
            }
          />
        )}
        <footer className="flex justify-end">
          <Button onClick={() => console.info(empty ? "Finish" : "Continue")}>
            {empty ? copy.wizardFinish : copy.wizardContinue}
          </Button>
        </footer>
      </PRFeedbackSetupWizardShell>
    </div>
  );
}
const meta = {
  title: "Factories/Configure branding",
  component: ConfigurePreview,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof ConfigurePreview>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Empty: Story = {};
export const Dark: Story = { globals: { theme: "dark" } };
export const Checks: Story = { args: { empty: false } };
export const ChecksDark: Story = { args: { empty: false }, globals: { theme: "dark" } };
export const Mobile: Story = { globals: { viewport: { value: "mobile1", isRotated: false } } };
