import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import { ConfigurationFieldModal } from "./ConfigurationFieldModal";
import { ComponentsConfigurationField } from "@/api-client";
import { Button } from "@/components/ui/button";

const meta = {
  title: "ui/ConfigurationFieldModal",
  component: ConfigurationFieldModal,
  parameters: {
    layout: "centered",
  },
  argTypes: {},
} satisfies Meta<typeof ConfigurationFieldModal>;

export default meta;

type Story = StoryObj<typeof ConfigurationFieldModal>;

export const AddNewField: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Add Field Modal</Button>
        <ConfigurationFieldModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          onSave={(field) => {
            console.log("Saved field:", field);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};

export const EditStringField: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    const existingField: ComponentsConfigurationField = {
      name: "api_key",
      label: "API Key",
      type: "string",
      description: "Your API key for authentication",
      required: true,
    };

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Edit String Field Modal</Button>
        <ConfigurationFieldModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          field={existingField}
          onSave={(field) => {
            console.log("Updated field:", field);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};

export const EditSelectField: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    const existingField: ComponentsConfigurationField = {
      name: "environment",
      label: "Environment",
      type: "select",
      description: "Target deployment environment",
      required: true,
      typeOptions: {
        select: {
          options: [
            { label: "Development", value: "dev" },
            { label: "Staging", value: "staging" },
            { label: "Production", value: "prod" },
          ],
        },
      },
    };

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Edit Select Field Modal</Button>
        <ConfigurationFieldModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          field={existingField}
          onSave={(field) => {
            console.log("Updated field:", field);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};

export const EditMultiSelectField: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    const existingField: ComponentsConfigurationField = {
      name: "regions",
      label: "Regions",
      type: "multi_select",
      description: "Select target regions for deployment",
      required: false,
      typeOptions: {
        multiSelect: {
          options: [
            { label: "US East", value: "us-east" },
            { label: "US West", value: "us-west" },
            { label: "EU Central", value: "eu-central" },
            { label: "Asia Pacific", value: "ap" },
          ],
        },
      },
    };

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Edit Multi-Select Field Modal</Button>
        <ConfigurationFieldModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          field={existingField}
          onSave={(field) => {
            console.log("Updated field:", field);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};

export const EditBooleanField: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    const existingField: ComponentsConfigurationField = {
      name: "auto_deploy",
      label: "Auto Deploy",
      type: "boolean",
      description: "Automatically deploy after approval",
      required: false,
    };

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Edit Boolean Field Modal</Button>
        <ConfigurationFieldModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          field={existingField}
          onSave={(field) => {
            console.log("Updated field:", field);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};

export const EditNumberField: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(false);
    const existingField: ComponentsConfigurationField = {
      name: "timeout",
      label: "Timeout",
      type: "number",
      description: "Timeout in seconds",
      required: true,
    };

    return (
      <div>
        <Button onClick={() => setIsOpen(true)}>Open Edit Number Field Modal</Button>
        <ConfigurationFieldModal
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          field={existingField}
          onSave={(field) => {
            console.log("Updated field:", field);
            setIsOpen(false);
          }}
        />
      </div>
    );
  },
};
