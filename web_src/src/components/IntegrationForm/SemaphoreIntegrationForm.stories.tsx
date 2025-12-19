import React, { useState, useRef } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { SemaphoreIntegrationForm } from "./SemaphoreIntegrationForm";
import type { IntegrationData, FormErrors } from "./types";

const meta: Meta<typeof SemaphoreIntegrationForm> = {
  title: "Components/IntegrationForm/SemaphoreIntegrationForm",
  component: SemaphoreIntegrationForm,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <div className="w-[600px] p-6 bg-white dark:bg-gray-900 rounded-lg">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: "",
      name: "",
      apiToken: {
        secretName: "",
        secretKey: "",
      },
    });

    const [errors, setErrors] = useState<FormErrors>({});
    const [secretValue, setSecretValue] = useState("");
    const orgUrlRef = useRef<HTMLInputElement>(null);

    return (
      <SemaphoreIntegrationForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secretValue={secretValue}
        setSecretValue={setSecretValue}
        orgUrlRef={orgUrlRef}
        {...args}
      />
    );
  },
};

export const WithExistingData: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: "https://myorg.semaphoreci.com",
      name: "myorg-organization",
      apiToken: {
        secretName: "",
        secretKey: "",
      },
    });

    const [errors, setErrors] = useState<FormErrors>({});
    const [secretValue, setSecretValue] = useState("");
    const orgUrlRef = useRef<HTMLInputElement>(null);

    return (
      <SemaphoreIntegrationForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secretValue={secretValue}
        setSecretValue={setSecretValue}
        orgUrlRef={orgUrlRef}
        {...args}
      />
    );
  },
};

export const WithErrors: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: "invalid-url",
      name: "",
      apiToken: {
        secretName: "",
        secretKey: "",
      },
    });

    const [errors, setErrors] = useState<FormErrors>({
      orgUrl: "URL must be a valid Semaphore organization URL",
      name: "Field cannot be empty",
    });
    const [secretValue, setSecretValue] = useState("");
    const orgUrlRef = useRef<HTMLInputElement>(null);

    return (
      <SemaphoreIntegrationForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secretValue={secretValue}
        setSecretValue={setSecretValue}
        orgUrlRef={orgUrlRef}
        {...args}
      />
    );
  },
};

export const WithApiEndpoint: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: "https://api.semaphoreci.com",
      name: "api-integration",
      apiToken: {
        secretName: "",
        secretKey: "",
      },
    });

    const [errors, setErrors] = useState<FormErrors>({});
    const [secretValue, setSecretValue] = useState("");
    const orgUrlRef = useRef<HTMLInputElement>(null);

    return (
      <SemaphoreIntegrationForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secretValue={secretValue}
        setSecretValue={setSecretValue}
        orgUrlRef={orgUrlRef}
        {...args}
      />
    );
  },
};

export const EmptyState: Story = {
  render: (args) => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: "",
      name: "",
      apiToken: {
        secretName: "",
        secretKey: "",
      },
    });

    const [errors, setErrors] = useState<FormErrors>({});
    const [secretValue, setSecretValue] = useState("");
    const orgUrlRef = useRef<HTMLInputElement>(null);

    return (
      <SemaphoreIntegrationForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secretValue={secretValue}
        setSecretValue={setSecretValue}
        orgUrlRef={orgUrlRef}
        {...args}
      />
    );
  },
};
