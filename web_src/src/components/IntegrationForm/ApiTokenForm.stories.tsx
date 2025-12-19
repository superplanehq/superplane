import React, { useState, useRef } from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiTokenForm } from "./ApiTokenForm";
import type { IntegrationData, FormErrors } from "./types";
import { defaultProps } from "../../../test/__mocks__/secrets";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
});

const meta: Meta<typeof ApiTokenForm> = {
  title: "Components/IntegrationForm/ApiTokenForm",
  component: ApiTokenForm,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <div className="w-[600px] p-6 bg-white dark:bg-gray-900 rounded-lg">
            <Story />
          </div>
        </MemoryRouter>
      </QueryClientProvider>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => {
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
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secretValue={secretValue}
        setSecretValue={setSecretValue}
        orgUrlRef={orgUrlRef}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    );
  },
};

export const WithSecretValue: Story = {
  render: () => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: "https://api.semaphoreci.com",
      name: "semaphore-integration",
      apiToken: {
        secretName: "",
        secretKey: "",
      },
    });

    const [errors, setErrors] = useState<FormErrors>({});
    const [secretValue, setSecretValue] = useState("smp_1234567890abcdef1234567890abcdef");
    const orgUrlRef = useRef<HTMLInputElement>(null);

    return (
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secretValue={secretValue}
        setSecretValue={setSecretValue}
        orgUrlRef={orgUrlRef}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    );
  },
};

export const WithErrors: Story = {
  render: () => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: "https://github.com/myorg",
      name: "myorg-account",
      apiToken: {
        secretName: "",
        secretKey: "",
      },
    });

    const [errors, setErrors] = useState<FormErrors>({
      secretValue: "Field cannot be empty",
    });
    const [secretValue, setSecretValue] = useState("");
    const orgUrlRef = useRef<HTMLInputElement>(null);

    return (
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secretValue={secretValue}
        setSecretValue={setSecretValue}
        orgUrlRef={orgUrlRef}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    );
  },
};

export const EditMode: Story = {
  render: () => {
    const [integrationData, setIntegrationData] = useState<IntegrationData>({
      orgUrl: "https://api.semaphoreci.com",
      name: "semaphore-integration",
      apiToken: {
        secretName: "semaphore-api-key",
        secretKey: "api-token",
      },
    });

    const [errors, setErrors] = useState<FormErrors>({});
    const [secretValue, setSecretValue] = useState("");
    const orgUrlRef = useRef<HTMLInputElement>(null);

    return (
      <ApiTokenForm
        integrationData={integrationData}
        setIntegrationData={setIntegrationData}
        errors={errors}
        setErrors={setErrors}
        secretValue={secretValue}
        setSecretValue={setSecretValue}
        orgUrlRef={orgUrlRef}
        isEditMode={true}
        organizationId={defaultProps.organizationId}
        canvasId={defaultProps.canvasId}
      />
    );
  },
};
