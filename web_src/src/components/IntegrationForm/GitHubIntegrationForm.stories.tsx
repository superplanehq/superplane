import React, { useState, useRef } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { GitHubIntegrationForm } from "./GitHubIntegrationForm";
import type { IntegrationData, FormErrors } from "./types";

const meta: Meta<typeof GitHubIntegrationForm> = {
  title: "Components/IntegrationForm/GitHubIntegrationForm",
  component: GitHubIntegrationForm,
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
      <GitHubIntegrationForm
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
      orgUrl: "https://github.com/myorg",
      name: "myorg-account",
      apiToken: {
        secretName: "",
        secretKey: "",
      },
    });

    const [errors, setErrors] = useState<FormErrors>({});
    const [secretValue, setSecretValue] = useState("");
    const orgUrlRef = useRef<HTMLInputElement>(null);

    return (
      <GitHubIntegrationForm
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
      orgUrl: "invalid-org-name!@#",
      name: "",
      apiToken: {
        secretName: "",
        secretKey: "",
      },
    });

    const [errors, setErrors] = useState<FormErrors>({
      orgUrl: "Invalid organization name. Only letters, numbers, and hyphens are allowed",
      name: "Field cannot be empty",
    });
    const [secretValue, setSecretValue] = useState("");
    const orgUrlRef = useRef<HTMLInputElement>(null);

    return (
      <GitHubIntegrationForm
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
      <GitHubIntegrationForm
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
