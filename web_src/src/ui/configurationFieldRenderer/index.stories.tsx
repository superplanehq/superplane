import React, { useMemo, useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { TooltipProvider } from "../../components/ui/tooltip";
import { ConfigurationFieldRenderer } from "./index";
import type { SuggestFieldValueFn } from "./types";
import {
  ConfigurationStorySeed,
  STORY_AUTOCOMPLETE_CONTEXT,
  STORY_DOMAIN_ID,
  STORY_DOMAIN_TYPE,
  STORY_INTEGRATION_ID,
  rendererCategoryOrder,
  rendererExampleMap,
  rendererExamples,
  type RendererExample,
} from "./storybooks/fixtures";

const meta = {
  title: "ui/ConfigurationFieldRenderer",
  tags: ["autodocs"],
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "A Storybook catalog for every field renderer routed by `ConfigurationFieldRenderer`. The stories mirror the type mapping in `pkg/configuration/field.go` and `web_src/src/ui/configurationFieldRenderer/index.tsx`, including the renderer-only `url` route.",
      },
    },
  },
  decorators: [
    (Story) => (
      <ConfigurationStorySeed>
        <TooltipProvider delayDuration={150}>
          <div className="max-w-5xl">
            <Story />
          </div>
        </TooltipProvider>
      </ConfigurationStorySeed>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

function RendererPlayground({
  example,
  assistantEnabled,
  suggestFieldValue,
}: {
  example: RendererExample;
  assistantEnabled?: boolean;
  suggestFieldValue?: SuggestFieldValueFn;
}) {
  const [value, setValue] = useState<unknown>(example.initialValue);
  const allValues = useMemo(() => {
    const fieldName = example.field.name;
    if (!fieldName) {
      return example.allValues ?? {};
    }

    return {
      ...(example.allValues ?? {}),
      [fieldName]: value,
    };
  }, [example.allValues, example.field.name, value]);

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
        <div className="mb-4 flex flex-wrap items-center gap-2">
          <span className="rounded-full bg-gray-900 px-2.5 py-1 text-xs font-semibold text-white">
            {example.field.type}
          </span>
          <span className="rounded-full border border-gray-200 px-2.5 py-1 text-xs text-gray-600">
            {example.source}
          </span>
          <span className="rounded-full border border-blue-200 bg-blue-50 px-2.5 py-1 text-xs text-blue-700">
            Go: {example.goType}
          </span>
        </div>
        <ConfigurationFieldRenderer
          allowExpressions={example.allowExpressions ?? false}
          field={example.field}
          value={value}
          onChange={setValue}
          allValues={allValues}
          domainId={STORY_DOMAIN_ID}
          domainType={STORY_DOMAIN_TYPE}
          organizationId={STORY_DOMAIN_ID}
          integrationId={STORY_INTEGRATION_ID}
          autocompleteExampleObj={STORY_AUTOCOMPLETE_CONTEXT}
          assistantEnabled={assistantEnabled}
          suggestFieldValue={suggestFieldValue}
        />
      </div>

      <div className="rounded-xl border border-gray-200 bg-gray-50 p-4">
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">Current value</p>
        <pre className="overflow-x-auto text-xs text-gray-700">{JSON.stringify(value, null, 2)}</pre>
      </div>
    </div>
  );
}

function RendererCatalog() {
  const [values, setValues] = useState<Record<string, unknown>>(() =>
    Object.fromEntries(
      rendererExamples
        .filter((example) => example.field.name)
        .map((example) => [example.field.name!, example.initialValue]),
    ),
  );

  return (
    <div className="space-y-8">
      {rendererCategoryOrder.map((category) => {
        const categoryExamples = rendererExamples.filter((example) => example.category === category);

        return (
          <section key={category} className="space-y-4">
            <div className="space-y-1">
              <h2 className="text-lg font-semibold text-gray-900">{category}</h2>
              <p className="text-sm text-gray-600">
                Stories in this section are grouped by how the renderer is typically used in component configuration.
              </p>
            </div>

            <div className="space-y-4">
              {categoryExamples.map((example) => {
                const fieldName = example.field.name!;
                const value = values[fieldName];
                const allValues = {
                  ...(example.allValues ?? {}),
                  [fieldName]: value,
                };

                return (
                  <div key={example.id} className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
                    <div className="mb-3 space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="text-sm font-semibold text-gray-900">{example.field.label}</h3>
                        <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700">
                          {example.field.type}
                        </span>
                        <span className="rounded-full border border-gray-200 px-2 py-0.5 text-xs text-gray-500">
                          {example.source}
                        </span>
                      </div>
                      <p className="text-xs font-mono text-blue-700">Go type: {example.goType}</p>
                      <p className="text-sm text-gray-600">{example.docsDescription}</p>
                    </div>

                    <ConfigurationFieldRenderer
                      allowExpressions={example.allowExpressions ?? false}
                      field={example.field}
                      value={value}
                      onChange={(nextValue) =>
                        setValues((currentValues) => ({
                          ...currentValues,
                          [fieldName]: nextValue,
                        }))
                      }
                      allValues={allValues}
                      domainId={STORY_DOMAIN_ID}
                      domainType={STORY_DOMAIN_TYPE}
                      organizationId={STORY_DOMAIN_ID}
                      integrationId={STORY_INTEGRATION_ID}
                      autocompleteExampleObj={STORY_AUTOCOMPLETE_CONTEXT}
                    />
                  </div>
                );
              })}
            </div>
          </section>
        );
      })}
    </div>
  );
}

function createExampleStory(exampleId: string): Story {
  const example = rendererExampleMap[exampleId];

  return {
    name: example.storyName.replace(/Field$/, ""),
    parameters: {
      docs: {
        description: {
          story: `${example.docsDescription} Go type: ${example.goType}.`,
        },
      },
    },
    render: () => <RendererPlayground example={example} />,
  };
}

export const Catalog: Story = {
  parameters: {
    docs: {
      description: {
        story:
          "A grouped overview for the full renderer surface area, including context-backed inputs and the compatibility `url` route.",
      },
    },
  },
  render: () => <RendererCatalog />,
};

export const StringField = createExampleStory("string");
export const TextField = createExampleStory("text");
export const ExpressionField = createExampleStory("expression");
export const XMLField = createExampleStory("xml");
export const NumberField = createExampleStory("number");
export const BooleanField = createExampleStory("boolean");
export const SelectField = createExampleStory("select");
export const MultiSelectField = createExampleStory("multi-select");
export const ListField = createExampleStory("list");
export const ObjectField = createExampleStory("object");
export const TimeField = createExampleStory("time");
export const DateField = createExampleStory("date");
export const DateTimeField = createExampleStory("datetime");
export const TimezoneField = createExampleStory("timezone");
export const DaysOfWeekField = createExampleStory("days-of-week");
export const TimeRangeField = createExampleStory("time-range");
export const DayInYearField = createExampleStory("day-in-year");
export const CronField = createExampleStory("cron");
export const UserField = createExampleStory("user");
export const RoleField = createExampleStory("role");
export const GroupField = createExampleStory("group");
export const IntegrationResourceField = createExampleStory("integration-resource");
export const AnyPredicateListField = createExampleStory("any-predicate-list");
export const GitRefField = createExampleStory("git-ref");
export const SecretKeyField = createExampleStory("secret-key");
export const UrlField = createExampleStory("url");
