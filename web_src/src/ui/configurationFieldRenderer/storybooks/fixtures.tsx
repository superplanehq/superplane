import React, { useRef } from "react";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import type {
  AuthorizationDomainType,
  ComponentsIntegrationRef,
  ConfigurationField,
  OrganizationsIntegration,
} from "@/api-client";
import { integrationKeys } from "@/hooks/useIntegrations";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { secretKeys } from "@/hooks/useSecrets";

export type RendererCategory =
  | "Basic Inputs"
  | "Structured Content"
  | "Date & Scheduling"
  | "Context-Aware Inputs"
  | "Compatibility";

export interface RendererExample {
  id: string;
  storyName: string;
  category: RendererCategory;
  source: "Basic field type" | "Special field type" | "Renderer-only route";
  goType: string;
  docsDescription: string;
  field: ConfigurationField;
  initialValue: unknown;
  allValues?: Record<string, unknown>;
  allowExpressions?: boolean;
}

export const STORY_DOMAIN_ID = "org_storybook";
export const STORY_DOMAIN_TYPE: AuthorizationDomainType = "DOMAIN_TYPE_ORGANIZATION";
export const STORY_ORGANIZATION_ID = STORY_DOMAIN_ID;
export const STORY_INTEGRATION_ID = "int_github_primary";
export const STORY_INTEGRATION_REF: ComponentsIntegrationRef = {
  id: STORY_INTEGRATION_ID,
  name: "GitHub Production",
};

export const STORY_INTEGRATIONS: OrganizationsIntegration[] = [
  {
    metadata: {
      id: STORY_INTEGRATION_ID,
      name: "GitHub Production",
      createdAt: "2026-03-30T10:00:00Z",
      updatedAt: "2026-03-30T10:00:00Z",
    },
    spec: {
      integrationName: "github",
    },
    status: {
      state: "ready",
      stateDescription: "The connection is ready to list repositories and other resources.",
    },
  },
];

export const STORY_AUTOCOMPLETE_CONTEXT = {
  trigger: {
    payload: {
      issue: {
        id: 1245,
        title: "Upgrade renderer stories",
        labels: ["frontend", "storybook"],
      },
      repository: {
        name: "core-api",
        defaultBranch: "main",
      },
    },
  },
  previousNode: {
    result: {
      url: "https://api.superplane.ai/webhooks/github",
      branch: "release/2026.04",
    },
  },
};

const baseField = (field: ConfigurationField): ConfigurationField => ({
  required: false,
  ...field,
});

export const rendererExamples: RendererExample[] = [
  {
    id: "string",
    storyName: "StringField",
    category: "Basic Inputs",
    source: "Basic field type",
    goType: "FieldTypeString",
    docsDescription:
      "Use `string` for short single-line values such as names, identifiers, subjects, branch names, and other compact freeform text.",
    field: baseField({
      name: "serviceName",
      label: "Service name",
      type: "string",
      placeholder: "Incident router",
      description: "Use for short single-line text like labels, IDs, or names that do not need a multiline editor.",
      required: true,
    }),
    initialValue: "Incident router",
    allowExpressions: true,
  },
  {
    id: "text",
    storyName: "TextField",
    category: "Structured Content",
    source: "Basic field type",
    goType: "FieldTypeText",
    docsDescription:
      "Use `text` when the value is still plain text, but users need a larger editor for prompts, long instructions, request bodies, or templates.",
    field: baseField({
      name: "promptBody",
      label: "Prompt body",
      type: "text",
      description:
        "Use for long-form plain text where a larger editor is easier to work with than a single input line.",
    }),
    initialValue:
      "Summarize the latest deployment signal.\nInclude risks, rollback steps, and the next operator action.",
  },
  {
    id: "expression",
    storyName: "ExpressionField",
    category: "Structured Content",
    source: "Basic field type",
    goType: "FieldTypeExpression",
    docsDescription:
      "Use `expression` when the field is intended to hold a computed value, template expression, or variable reference rather than only fixed text.",
    field: baseField({
      name: "subjectTemplate",
      label: "Subject template",
      type: "expression",
      placeholder: '$["trigger"].payload.issue.title',
      description:
        "Use for computed values and templates that are expected to reference workflow data instead of only fixed text.",
    }),
    initialValue: '$["trigger"].payload.issue.title',
    allowExpressions: true,
  },
  {
    id: "xml",
    storyName: "XMLField",
    category: "Structured Content",
    source: "Basic field type",
    goType: "FieldTypeXML",
    docsDescription:
      "Use `xml` for integrations or APIs that expect XML documents, SOAP envelopes, or XML templates that benefit from syntax-aware editing.",
    field: baseField({
      name: "soapBody",
      label: "SOAP body",
      type: "xml",
      description: "Use when the receiving API expects an XML payload and authors benefit from a structured editor.",
    }),
    initialValue: "<Envelope><Body><CreateTicket><Title>Renderer coverage</Title></CreateTicket></Body></Envelope>",
  },
  {
    id: "number",
    storyName: "NumberField",
    category: "Basic Inputs",
    source: "Basic field type",
    goType: "FieldTypeNumber",
    docsDescription:
      "Use `number` for thresholds, retry counts, durations, limits, and any numeric input with optional min and max constraints.",
    field: baseField({
      name: "retryCount",
      label: "Retry count",
      type: "number",
      description: "Use for numeric settings like limits, delays, percentages, or retry policies.",
      typeOptions: {
        number: {
          min: 0,
          max: 10,
        },
      },
    }),
    initialValue: 3,
  },
  {
    id: "boolean",
    storyName: "BooleanField",
    category: "Basic Inputs",
    source: "Basic field type",
    goType: "FieldTypeBool",
    docsDescription:
      "Use `boolean` for binary on/off choices where the user is enabling or disabling a feature, condition, or optional behavior.",
    field: baseField({
      name: "sendDigest",
      label: "Send digest",
      type: "boolean",
      description: "Use for binary feature flags and other simple enabled or disabled behavior.",
    }),
    initialValue: true,
  },
  {
    id: "select",
    storyName: "SelectField",
    category: "Basic Inputs",
    source: "Basic field type",
    goType: "FieldTypeSelect",
    docsDescription:
      "Use `select` when the user must choose exactly one value from a fixed list declared in the field definition.",
    field: baseField({
      name: "environment",
      label: "Environment",
      type: "select",
      description: "Use when the valid values are known upfront and only one option can be chosen.",
      typeOptions: {
        select: {
          options: [
            { label: "Development", value: "development" },
            { label: "Staging", value: "staging" },
            { label: "Production", value: "production" },
          ],
        },
      },
    }),
    initialValue: "production",
  },
  {
    id: "multi-select",
    storyName: "MultiSelectField",
    category: "Basic Inputs",
    source: "Basic field type",
    goType: "FieldTypeMultiSelect",
    docsDescription:
      "Use `multi-select` when the field accepts several values from a fixed list and each selected value should remain individually visible.",
    field: baseField({
      name: "deliveryChannels",
      label: "Delivery channels",
      type: "multi-select",
      description: "Use when several predefined options can be selected at the same time.",
      typeOptions: {
        multiSelect: {
          options: [
            { label: "Email", value: "email" },
            { label: "Slack", value: "slack" },
            { label: "PagerDuty", value: "pagerduty" },
            { label: "Webhook", value: "webhook" },
          ],
        },
      },
    }),
    initialValue: ["slack", "webhook"],
  },
  {
    id: "list",
    storyName: "ListField",
    category: "Structured Content",
    source: "Basic field type",
    goType: "FieldTypeList",
    docsDescription:
      "Use `list` when a setting accepts zero-to-many repeated items, either primitive values or nested object entries with their own schema.",
    field: baseField({
      name: "headers",
      label: "Headers",
      type: "list",
      description:
        "Use when users can add any number of repeated items, especially small object rows such as headers or mappings.",
      typeOptions: {
        list: {
          itemLabel: "Header",
          itemDefinition: {
            type: "object",
            schema: [
              {
                name: "key",
                label: "Key",
                type: "string",
                required: true,
              },
              {
                name: "value",
                label: "Value",
                type: "string",
                required: true,
              },
            ],
          },
        },
      },
    }),
    initialValue: [
      { key: "X-Environment", value: "production" },
      { key: "X-Request-Source", value: "storybook" },
    ],
    allowExpressions: true,
  },
  {
    id: "object",
    storyName: "ObjectField",
    category: "Structured Content",
    source: "Basic field type",
    goType: "FieldTypeObject",
    docsDescription:
      "Use `object` when several related settings belong together under a single nested value and should be edited as a sub-form.",
    field: baseField({
      name: "authConfig",
      label: "Authentication",
      type: "object",
      description:
        "Use when a single setting owns a nested schema of related fields such as auth, formatting, or advanced options.",
      typeOptions: {
        object: {
          schema: [
            {
              name: "authMethod",
              label: "Auth method",
              type: "select",
              required: true,
              typeOptions: {
                select: {
                  options: [
                    { label: "API token", value: "token" },
                    { label: "Basic auth", value: "basic" },
                  ],
                },
              },
            },
            {
              name: "token",
              label: "Token",
              type: "string",
              sensitive: true,
              visibilityConditions: [{ field: "authMethod", values: ["token"] }],
            },
            {
              name: "username",
              label: "Username",
              type: "string",
              visibilityConditions: [{ field: "authMethod", values: ["basic"] }],
            },
            {
              name: "password",
              label: "Password",
              type: "string",
              sensitive: true,
              visibilityConditions: [{ field: "authMethod", values: ["basic"] }],
            },
            {
              name: "includeMetadata",
              label: "Include metadata",
              type: "boolean",
            },
          ],
        },
      },
    }),
    initialValue: {
      authMethod: "token",
      token: "sp_live_token",
      includeMetadata: true,
    },
  },
  {
    id: "time",
    storyName: "TimeField",
    category: "Date & Scheduling",
    source: "Basic field type",
    goType: "FieldTypeTime",
    docsDescription:
      "Use `time` for a time of day without any attached date, typically for cutoffs, business windows, or recurring daily schedules.",
    field: baseField({
      name: "runAt",
      label: "Run at",
      type: "time",
      description: "Use for a time-of-day value when no specific calendar date is part of the setting.",
      typeOptions: {
        time: {
          format: "HH:MM",
        },
      },
    }),
    initialValue: "09:30",
  },
  {
    id: "date",
    storyName: "DateField",
    category: "Date & Scheduling",
    source: "Basic field type",
    goType: "FieldTypeDate",
    docsDescription:
      "Use `date` when the setting is a calendar day only, such as a launch date, expiration date, or one-time schedule anchor.",
    field: baseField({
      name: "goLiveDate",
      label: "Go-live date",
      type: "date",
      description: "Use for calendar dates when the time of day is not relevant.",
    }),
    initialValue: "2026-04-15",
  },
  {
    id: "datetime",
    storyName: "DateTimeField",
    category: "Date & Scheduling",
    source: "Basic field type",
    goType: "FieldTypeDateTime",
    docsDescription:
      "Use `datetime` when both a calendar date and a local time are required and the field should be edited together.",
    field: baseField({
      name: "freezeAt",
      label: "Freeze at",
      type: "datetime",
      description: "Use when a setting needs a combined date and time instead of two separate inputs.",
    }),
    initialValue: "2026-04-15T13:30",
  },
  {
    id: "timezone",
    storyName: "TimezoneField",
    category: "Date & Scheduling",
    source: "Basic field type",
    goType: "FieldTypeTimezone",
    docsDescription:
      "Use `timezone` when another scheduled value needs an explicit timezone offset so the system can interpret it consistently.",
    field: baseField({
      name: "scheduleTimezone",
      label: "Schedule timezone",
      type: "timezone",
      description: "Use to define which timezone a time or schedule should be interpreted in.",
    }),
    initialValue: "-3",
  },
  {
    id: "days-of-week",
    storyName: "DaysOfWeekField",
    category: "Date & Scheduling",
    source: "Basic field type",
    goType: "FieldTypeDaysOfWeek",
    docsDescription:
      "Use `days-of-week` for recurring weekly schedules where the user picks one or more weekdays directly.",
    field: baseField({
      name: "runDays",
      label: "Run days",
      type: "days-of-week",
      description: "Use for recurring weekly schedules where users should pick one or more weekdays directly.",
    }),
    initialValue: ["monday", "wednesday", "friday"],
  },
  {
    id: "time-range",
    storyName: "TimeRangeField",
    category: "Date & Scheduling",
    source: "Basic field type",
    goType: "FieldTypeTimeRange",
    docsDescription:
      "Use `time-range` when the configuration represents a start and end window inside a single day, such as business hours or blackout periods.",
    field: baseField({
      name: "officeHours",
      label: "Office hours",
      type: "time-range",
      description: "Use for a start and end window inside one day, such as quiet hours or support coverage.",
    }),
    initialValue: "08:00 - 18:00",
  },
  {
    id: "day-in-year",
    storyName: "DayInYearField",
    category: "Date & Scheduling",
    source: "Special field type",
    goType: "FieldTypeDayInYear",
    docsDescription:
      "Use `day-in-year` for annual recurrences that care about month and day, but not the year, like renewals, anniversaries, or holidays.",
    field: baseField({
      name: "renewalDate",
      label: "Renewal date",
      type: "day-in-year",
      description: "Use for annual recurrences where only the month and day matter.",
    }),
    initialValue: "09/15",
  },
  {
    id: "cron",
    storyName: "CronField",
    category: "Date & Scheduling",
    source: "Special field type",
    goType: "FieldTypeCron",
    docsDescription:
      "Use `cron` when the schedule is complex enough that power users should supply a cron expression instead of simpler scheduling controls.",
    field: baseField({
      name: "cronSchedule",
      label: "Cron schedule",
      type: "cron",
      placeholder: "30 14 * * MON-FRI",
      description: "Use for advanced recurring schedules that need the flexibility of a cron expression.",
    }),
    initialValue: "30 14 * * MON-FRI",
  },
  {
    id: "user",
    storyName: "UserField",
    category: "Context-Aware Inputs",
    source: "Special field type",
    goType: "FieldTypeUser",
    docsDescription:
      "Use `user` when the setting must point to a concrete organization user, such as an assignee, owner, or explicit approver.",
    field: baseField({
      name: "ownerUserId",
      label: "Owner",
      type: "user",
      description: "Use when the setting must reference a specific user from the current organization.",
    }),
    initialValue: "user_2",
  },
  {
    id: "role",
    storyName: "RoleField",
    category: "Context-Aware Inputs",
    source: "Special field type",
    goType: "FieldTypeRole",
    docsDescription:
      "Use `role` when the config should target a reusable organization role instead of one specific user.",
    field: baseField({
      name: "approvalRole",
      label: "Approval role",
      type: "role",
      description: "Use when any member of a role can satisfy the configuration instead of a fixed user.",
    }),
    initialValue: "incident-commander",
  },
  {
    id: "group",
    storyName: "GroupField",
    category: "Context-Aware Inputs",
    source: "Special field type",
    goType: "FieldTypeGroup",
    docsDescription: "Use `group` when approvals, notifications, or routing should target a named organization group.",
    field: baseField({
      name: "escalationGroup",
      label: "Escalation group",
      type: "group",
      description: "Use when notifications or approvals should route to a named organization group.",
    }),
    initialValue: "platform-oncall",
  },
  {
    id: "integration-resource",
    storyName: "IntegrationResourceField",
    category: "Context-Aware Inputs",
    source: "Special field type",
    goType: "FieldTypeIntegrationResource",
    docsDescription:
      "Use `integration-resource` when the options must come from the connected integration instance, such as repositories, channels, projects, or boards.",
    field: baseField({
      name: "repository",
      label: "Repository",
      type: "integration-resource",
      description:
        "Use when options must be loaded from the connected integration instead of being hard-coded in the field definition.",
      typeOptions: {
        resource: {
          type: "repository",
          useNameAsValue: true,
        },
      },
    }),
    initialValue: "core-api",
    allowExpressions: true,
  },
  {
    id: "any-predicate-list",
    storyName: "AnyPredicateListField",
    category: "Structured Content",
    source: "Special field type",
    goType: "FieldTypeAnyPredicateList",
    docsDescription:
      "Use `any-predicate-list` when the value is a repeated list of operator and value checks that behave like OR-style match predicates.",
    field: baseField({
      name: "matchAny",
      label: "Match any",
      type: "any-predicate-list",
      description: "Use for repeated operator and value conditions when any one predicate can match the incoming data.",
      placeholder: "billing",
      typeOptions: {
        anyPredicateList: {
          operators: [
            { label: "Contains", value: "contains" },
            { label: "Equals", value: "equals" },
            { label: "Starts with", value: "starts_with" },
          ],
        },
      },
    }),
    initialValue: [
      { type: "contains", value: '{{ $["trigger"].payload.issue.title }}' },
      { type: "equals", value: "billing" },
    ],
    allowExpressions: true,
  },
  {
    id: "git-ref",
    storyName: "GitRefField",
    category: "Context-Aware Inputs",
    source: "Special field type",
    goType: "FieldTypeGitRef",
    docsDescription:
      "Use `git-ref` when downstream integrations need a normalized Git reference in `refs/heads/*` or `refs/tags/*` form.",
    field: baseField({
      name: "deployRef",
      label: "Deploy ref",
      type: "git-ref",
      description: "Use when the stored value must be a normalized Git branch or tag reference.",
    }),
    initialValue: "refs/heads/main",
  },
  {
    id: "secret-key",
    storyName: "SecretKeyField",
    category: "Context-Aware Inputs",
    source: "Special field type",
    goType: "FieldTypeSecretKey",
    docsDescription:
      "Use `secret-key` when the configuration should reference a stored credential key rather than collecting raw secret text in the form.",
    field: baseField({
      name: "apiCredential",
      label: "API credential",
      type: "secret-key",
      description:
        "Use when the component should reference an existing stored credential instead of plain text secrets.",
    }),
    initialValue: {
      secret: "deploy-credentials",
      key: "TOKEN",
    },
  },
  {
    id: "url",
    storyName: "UrlField",
    category: "Compatibility",
    source: "Renderer-only route",
    goType: "No matching Go constant",
    docsDescription:
      "Use `url` for fixed web endpoints. This renderer is routed in `ConfigurationFieldRenderer` even though `pkg/configuration/field.go` does not declare a `url` constant.",
    field: baseField({
      name: "callbackUrl",
      label: "Callback URL",
      type: "url",
      description:
        "Use for web endpoints and callback targets. This route exists in the renderer map even though it is not listed in the Go field constants.",
      placeholder: "https://api.superplane.ai/webhooks/github",
    }),
    initialValue: "https://api.superplane.ai/webhooks/github",
  },
];

export const rendererExampleMap = Object.fromEntries(
  rendererExamples.map((example) => [example.id, example]),
) as Record<string, RendererExample>;

export const rendererCategoryOrder: RendererCategory[] = [
  "Basic Inputs",
  "Structured Content",
  "Date & Scheduling",
  "Context-Aware Inputs",
  "Compatibility",
];

export const settingsTabFields: ConfigurationField[] = rendererExamples.map((example) => example.field);

export const settingsTabConfiguration = Object.fromEntries(
  rendererExamples
    .filter((example) => example.field.name)
    .map((example) => [example.field.name!, example.initialValue]),
);

const mockUsers = [
  {
    metadata: {
      id: "user_1",
      email: "ana@superplane.dev",
    },
    spec: {
      displayName: "Ana Martins",
    },
  },
  {
    metadata: {
      id: "user_2",
      email: "pedro@superplane.dev",
    },
    spec: {
      displayName: "Pedro Foresti Leao",
    },
  },
];

const mockRoles = [
  {
    metadata: {
      name: "incident-commander",
    },
    spec: {
      displayName: "Incident Commander",
    },
  },
  {
    metadata: {
      name: "release-manager",
    },
    spec: {
      displayName: "Release Manager",
    },
  },
];

const mockGroups = [
  {
    metadata: {
      name: "platform-oncall",
    },
    spec: {
      displayName: "Platform On-Call",
    },
  },
  {
    metadata: {
      name: "security-reviewers",
    },
    spec: {
      displayName: "Security Reviewers",
    },
  },
];

const mockIntegrationResources = [
  {
    id: "repo_1",
    name: "core-api",
    type: "repository",
  },
  {
    id: "repo_2",
    name: "web-app",
    type: "repository",
  },
  {
    id: "repo_3",
    name: "automation-hub",
    type: "repository",
  },
];

const mockSecrets = [
  {
    metadata: {
      id: "secret_deploy_credentials",
      name: "deploy-credentials",
      domainId: STORY_DOMAIN_ID,
      domainType: STORY_DOMAIN_TYPE,
    },
    spec: {
      provider: "PROVIDER_LOCAL",
    },
  },
  {
    metadata: {
      id: "secret_crm_credentials",
      name: "crm-credentials",
      domainId: STORY_DOMAIN_ID,
      domainType: STORY_DOMAIN_TYPE,
    },
    spec: {
      provider: "PROVIDER_LOCAL",
    },
  },
];

const mockSecretDetails = {
  "deploy-credentials": {
    metadata: {
      id: "secret_deploy_credentials",
      name: "deploy-credentials",
      domainId: STORY_DOMAIN_ID,
      domainType: STORY_DOMAIN_TYPE,
    },
    spec: {
      provider: "PROVIDER_LOCAL",
      local: {
        data: {
          TOKEN: "••••••••",
          WEBHOOK_SECRET: "••••••••",
        },
      },
    },
  },
  "crm-credentials": {
    metadata: {
      id: "secret_crm_credentials",
      name: "crm-credentials",
      domainId: STORY_DOMAIN_ID,
      domainType: STORY_DOMAIN_TYPE,
    },
    spec: {
      provider: "PROVIDER_LOCAL",
      local: {
        data: {
          API_KEY: "••••••••",
        },
      },
    },
  },
};

export function seedConfigurationStoryQueryCache(queryClient: QueryClient) {
  queryClient.setQueryData(organizationKeys.users(STORY_DOMAIN_ID), mockUsers);
  queryClient.setQueryData(organizationKeys.roles(STORY_DOMAIN_ID), mockRoles);
  queryClient.setQueryData(organizationKeys.groups(STORY_DOMAIN_ID), mockGroups);
  queryClient.setQueryData(
    integrationKeys.resources(STORY_ORGANIZATION_ID, STORY_INTEGRATION_ID, "repository"),
    mockIntegrationResources,
  );
  queryClient.setQueryData(secretKeys.byDomain(STORY_DOMAIN_ID, STORY_DOMAIN_TYPE), mockSecrets);

  Object.entries(mockSecretDetails).forEach(([secretRef, secret]) => {
    queryClient.setQueryData(secretKeys.detail(STORY_DOMAIN_ID, STORY_DOMAIN_TYPE, secretRef), secret);
  });
}

export function ConfigurationStorySeed({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const hasSeeded = useRef(false);

  if (!hasSeeded.current) {
    seedConfigurationStoryQueryCache(queryClient);
    hasSeeded.current = true;
  }

  return <>{children}</>;
}
