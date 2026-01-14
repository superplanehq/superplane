import type { Meta, StoryObj } from "@storybook/react";
import { AutoCompleteInput } from "./AutoCompleteInput";

const meta: Meta<typeof AutoCompleteInput> = {
  title: "Components/AutoCompleteInput",
  component: AutoCompleteInput,
  parameters: {
    layout: "centered",
  },
  decorators: [
    (Story) => (
      <div className="w-[520px] max-w-full">
        <Story />
      </div>
    ),
  ],
  tags: ["autodocs"],
  argTypes: {
    disabled: {
      control: "boolean",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const exprLangObject = {
  $: {
    data: {
      version: "1.2.3",
      service: "api",
      env: "prod",
    },
    deploy: {
      id: "dep_123",
      region: "us-east-1",
    },
  },
};

export const ExprLanguage: Story = {
  args: {
    exampleObj: exprLangObject,
    placeholder: "{{ $.data.version }} deployment",
    inputSize: "sm",
    startWord: "{",
    prefix: "{{ $.",
    suffix: " }}",
    showValuePreview: true,
  },
};
