import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";

import { FEATURE_ORGANIZATION_BYOK } from "@/lib/experimentalFeatures";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  FactorySettingsLLMModelsSwitchPreview,
  type LLMModelsSwitchDialog,
  type LLMModelsSwitchProvider,
} from "./FactorySettingsLLMModelsSwitchPreview";

const meta = {
  title: "Factories/Pages/Settings/LLM Models switch",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

const modelsPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/models`;

function SwitchFlow({
  initialSource = "hosted",
  initialDialog = null,
  initialNotice = false,
}: {
  initialSource?: "hosted" | LLMModelsSwitchProvider;
  initialDialog?: LLMModelsSwitchDialog;
  initialNotice?: boolean;
}) {
  const [source, setSource] = useState<"hosted" | LLMModelsSwitchProvider>(initialSource);
  const [dialog, setDialog] = useState<LLMModelsSwitchDialog>(initialDialog);
  const [switchedNotice, setSwitchedNotice] = useState(initialNotice);

  return (
    <FactorySettingsLLMModelsSwitchPreview
      source={source}
      dialog={dialog}
      switchedNotice={switchedNotice}
      onChoose={(target) => {
        console.log("choose model source", target);
        setDialog({ target, step: "warn" });
      }}
      onCancel={() => {
        console.log("cancel model source switch");
        setDialog(null);
      }}
      onContinueToKey={() => {
        console.log("continue to model source key");
        setDialog((current) => (current && current.target !== "hosted" ? { ...current, step: "key" } : current));
      }}
      onBackToWarning={() => {
        console.log("back to model source warning");
        setDialog((current) => (current ? { ...current, step: "warn" } : current));
      }}
      onSaveKey={() => {
        console.log("save model source key");
        if (!dialog || dialog.target === "hosted") {
          return;
        }
        setSource(dialog.target);
        setSwitchedNotice(true);
        setDialog(null);
      }}
      onSwitchToHosted={() => {
        console.log("switch model source to SuperPlane");
        setSource("hosted");
        setSwitchedNotice(true);
        setDialog(null);
      }}
    />
  );
}

function FullSettingsPage(props: {
  initialSource?: "hosted" | LLMModelsSwitchProvider;
  initialDialog?: LLMModelsSwitchDialog;
  initialNotice?: boolean;
}) {
  return (
    <FactoriesHarness
      pathSuffix={modelsPath}
      experimentalFeatures={[FEATURE_ORGANIZATION_BYOK]}
      factoriesFixture={defaultFactoriesFixture}
      pageOverrides={{ llmModels: () => <SwitchFlow {...props} /> }}
    />
  );
}

export const HostedWithProviders: Story = {
  name: "SuperPlane, then connect",
  render: () => <FullSettingsPage />,
};

export const WarnBeforeSwitch: Story = {
  name: "Warn before switch",
  render: () => <FullSettingsPage initialDialog={{ target: "anthropic", step: "warn" }} />,
};

export const EnterKey: Story = {
  name: "Enter key",
  render: () => <FullSettingsPage initialDialog={{ target: "anthropic", step: "key" }} />,
};

export const SwitchedToClaude: Story = {
  name: "Switched to Claude",
  render: () => <FullSettingsPage initialSource="anthropic" initialNotice />,
};
