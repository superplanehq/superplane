import React, { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { TooltipProvider } from "@/ui/tooltip";
import { SettingsTab } from "./SettingsTab";
import {
  ConfigurationStorySeed,
  STORY_AUTOCOMPLETE_CONTEXT,
  STORY_DOMAIN_ID,
  STORY_DOMAIN_TYPE,
  STORY_INTEGRATION_REF,
  STORY_INTEGRATIONS,
  settingsTabConfiguration,
  settingsTabFields,
} from "@/ui/configurationFieldRenderer/storybooks/fixtures";

const meta = {
  title: "ui/ComponentSidebar/SettingsTab",
  component: SettingsTab,
  tags: ["autodocs"],
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "An aggregated `SettingsTab` story that renders the configuration field catalog through the same sidebar path used in the product, including realtime validation, integration context, and organization-backed field lookups.",
      },
    },
  },
  decorators: [
    (Story) => (
      <ConfigurationStorySeed>
        <TooltipProvider delayDuration={150}>
          <div className="w-[760px] max-w-full rounded-xl border border-gray-200 bg-white shadow-sm">
            <Story />
          </div>
        </TooltipProvider>
      </ConfigurationStorySeed>
    ),
  ],
} satisfies Meta<typeof SettingsTab>;

export default meta;

type Story = StoryObj<typeof meta>;

function SettingsTabPlayground() {
  const [configuration, setConfiguration] = useState<Record<string, unknown>>(settingsTabConfiguration);
  const [nodeName, setNodeName] = useState("Renderer Coverage Demo");
  const [integrationRef, setIntegrationRef] = useState(STORY_INTEGRATION_REF);

  return (
    <SettingsTab
      mode="edit"
      nodeId="node_renderer_coverage"
      nodeName={nodeName}
      configuration={configuration}
      configurationFields={settingsTabFields}
      onSave={(updatedConfiguration, updatedNodeName, updatedIntegrationRef) => {
        setConfiguration(updatedConfiguration);
        setNodeName(updatedNodeName);
        setIntegrationRef(updatedIntegrationRef);
        console.log("SettingsTab saved", {
          configuration: updatedConfiguration,
          nodeName: updatedNodeName,
          integration: updatedIntegrationRef,
        });
      }}
      domainId={STORY_DOMAIN_ID}
      domainType={STORY_DOMAIN_TYPE}
      integrationName="github"
      integrationRef={integrationRef}
      integrations={STORY_INTEGRATIONS}
      integrationDefinition={{
        name: "github",
        label: "GitHub",
        icon: "github",
      }}
      autocompleteExampleObj={STORY_AUTOCOMPLETE_CONTEXT}
      onOpenCreateIntegrationDialog={() => console.log("Open integration connect dialog")}
      onOpenConfigureIntegrationDialog={(integrationId) =>
        console.log("Open integration configuration dialog", integrationId)
      }
      configurationSaveMode="manual"
    />
  );
}

export const RendererCoverage: Story = {
  name: "Renderer Coverage",
  parameters: {
    docs: {
      description: {
        story:
          "Renders the same field catalog inside `SettingsTab` so renderer behavior can be reviewed in the full component-sidebar layout, not only as isolated fields.",
      },
    },
  },
  render: () => <SettingsTabPlayground />,
};
