import React, { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { AutoCompleteInput } from "./AutoCompleteInput";
import { STORY_AUTOCOMPLETE_CONTEXT } from "@/ui/configurationFieldRenderer/storybooks/fixtures";

function CanvasSettingsSidebar({ children }: { children: React.ReactNode }) {
  return (
    <div className="w-full max-w-[612px] border-l border-border bg-white">
      <div className="p-4">{children}</div>
    </div>
  );
}

function CanvasSettingsSection({ children }: { children: React.ReactNode }) {
  return (
    <div className="space-y-6">
      <div className="border-t border-gray-200 pt-6 space-y-4">{children}</div>
    </div>
  );
}

function CanvasListItemShell({ fieldLabel, children }: { fieldLabel: string; children: React.ReactNode }) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <div className="flex-1">
          <div className="rounded-md bg-slate-100 p-4 space-y-4">
            <div className="space-y-2">
              <div className="text-sm font-medium text-gray-800">{fieldLabel}</div>
              {children}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function FieldLabel({ children }: { children: React.ReactNode }) {
  return <label className="text-sm font-medium text-gray-800">{children}</label>;
}

const meta: Meta<typeof AutoCompleteInput> = {
  title: "Components/AutoCompleteInput",
  component: AutoCompleteInput,
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
  argTypes: {
    disabled: {
      control: "boolean",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/** Legacy wrapped-expression demo (smaller input size). */
export const ExprLanguage: Story = {
  decorators: [
    (Story) => (
      <div className="flex min-h-screen items-start justify-center bg-slate-100 p-8">
        <div className="w-[520px] max-w-full rounded-md border border-gray-200 bg-white p-4">
          <Story />
        </div>
      </div>
    ),
  ],
  args: {
    exampleObj: STORY_AUTOCOMPLETE_CONTEXT,
    placeholder: "{{ $.trigger.payload.issue.title }}",
    inputSize: "sm",
    startWord: "{",
    prefix: "{{ ",
    suffix: " }}",
    showValuePreview: true,
  },
};

/** Matches `ExpressionFieldRenderer` in canvas node settings (raw expressions). */
export const CanvasSettingsExpressionField: Story = {
  name: "Canvas Settings / Expression Field",
  render: function CanvasSettingsExpressionFieldStory() {
    const [value, setValue] = useState("int(now().Unix())");

    return (
      <div className="flex min-h-screen justify-end bg-slate-100">
        <CanvasSettingsSidebar>
          <CanvasSettingsSection>
            <div className="space-y-2">
              <FieldLabel>Field Value*</FieldLabel>
              <AutoCompleteInput
                exampleObj={STORY_AUTOCOMPLETE_CONTEXT}
                value={value}
                onChange={setValue}
                placeholder=""
                expressionMode="raw"
                inputSize="md"
                showValuePreview
                quickTip="Tip: type `$` to browse node payloads."
                data-testid="expression-field-value"
              />
            </div>
          </CanvasSettingsSection>
        </CanvasSettingsSidebar>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          'Mirrors expression fields in the canvas settings sidebar: raw expression mode, `inputSize="md"`, value preview, and the `$` quick tip. Focus the field and type `$` to open the suggestion menu.',
      },
    },
  },
};

/** Matches `StringFieldRenderer` with expressions inside a list item (`bg-slate-100` shell). */
export const CanvasSettingsListStringField: Story = {
  name: "Canvas Settings / List String Field",
  render: function CanvasSettingsListStringFieldStory() {
    const [name, setName] = useState("updated_at");
    const [value, setValue] = useState("{{ $.trigger.payload.issue.title }}");

    return (
      <div className="flex min-h-screen justify-end bg-slate-100">
        <CanvasSettingsSidebar>
          <CanvasSettingsSection>
            <div className="space-y-2">
              <FieldLabel>Fields</FieldLabel>
              <CanvasListItemShell fieldLabel="Field row">
                <div className="space-y-4">
                  <div className="space-y-2">
                    <FieldLabel>Field Name*</FieldLabel>
                    <AutoCompleteInput
                      exampleObj={STORY_AUTOCOMPLETE_CONTEXT}
                      value={name}
                      onChange={setName}
                      placeholder=""
                      startWord="{{"
                      prefix="{{ "
                      suffix=" }}"
                      inputSize="md"
                      showValuePreview
                      quickTip="Tip: type `{{` to start an expression."
                      data-testid="string-field-name"
                    />
                  </div>
                  <div className="space-y-2">
                    <FieldLabel>Field Value*</FieldLabel>
                    <AutoCompleteInput
                      exampleObj={STORY_AUTOCOMPLETE_CONTEXT}
                      value={value}
                      onChange={setValue}
                      placeholder=""
                      startWord="{{"
                      prefix="{{ "
                      suffix=" }}"
                      inputSize="md"
                      showValuePreview
                      quickTip="Tip: type `{{` to start an expression."
                      data-testid="string-field-value"
                    />
                  </div>
                </div>
              </CanvasListItemShell>
            </div>
          </CanvasSettingsSection>
        </CanvasSettingsSidebar>
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story:
          "Mirrors string fields with wrapped expressions inside list items on the canvas settings sidebar, including the slate list-item shell and white autocomplete wrapper.",
      },
    },
  },
};
