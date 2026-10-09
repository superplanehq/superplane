import type { Meta, StoryObj } from "@storybook/react-vite";
import { ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";

import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { DRAFT_READINESS_NOTES } from "../../lib/draftReadiness";
import { PlanningImplementationControls } from "./PlanningImplementationControls";

const meta = {
  title: "Factories/Pages/Task Split Run/Phone override",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [withFactoriesTheme],
} satisfies Meta;

export default meta;

type Story = StoryObj;

const MODEL_LABEL = "claude-opus-4-6 Medium";

export const Discouraged: Story = {
  render: () => (
    <div className="min-h-svh bg-background">
      <PlanningImplementationControls
        startDiscouraged
        readinessNote={DRAFT_READINESS_NOTES.uncertain}
        modelSelect={<PhoneModelControl />}
        actions={<PhoneStartAction />}
        canSend
        showSuggestChanges={false}
        onSuggestChanges={() => undefined}
      />
    </div>
  ),
};

function PhoneModelControl() {
  return (
    <Button
      type="button"
      size="sm"
      variant="outline"
      aria-label={`Model: ${MODEL_LABEL}`}
      data-testid="split-run-draft-model"
      className="h-7 max-w-36 min-w-0 shrink gap-1.5 overflow-hidden !rounded-none border-0 bg-background px-2 text-xs shadow-none"
    >
      <span className="min-w-0 truncate">{MODEL_LABEL}</span>
      <ChevronDown className="size-3 shrink-0 opacity-60" aria-hidden />
    </Button>
  );
}

function PhoneStartAction() {
  return (
    <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
      <Button type="button" size="sm" variant="outline">
        Start
      </Button>
    </div>
  );
}
