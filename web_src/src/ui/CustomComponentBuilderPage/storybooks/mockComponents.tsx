import { ComponentsComponent } from "@/api-client";

export const mockComponents: ComponentsComponent[] = [
  {
    name: "http",
    label: "HTTP Request",
    description: "Make HTTP requests to external services",
    icon: "globe",
    color: "blue",
    outputChannels: [
      { name: "default", label: "Success" },
      { name: "error", label: "Error" },
    ],
    configuration: [
      {
        name: "url",
        label: "URL",
        type: "string",
        required: true,
      },
      {
        name: "method",
        label: "Method",
        type: "select",
        required: true,
      },
    ],
  },
  {
    name: "if",
    label: "If",
    description: "Branch execution based on a condition",
    icon: "braces",
    color: "gray",
    outputChannels: [
      { name: "then", label: "Then" },
      { name: "else", label: "Else" },
    ],
    configuration: [
      {
        name: "expression",
        label: "Expression",
        type: "string",
        required: true,
      },
    ],
  },
  {
    name: "approval",
    label: "Approval",
    description: "Pause for manual approval",
    icon: "check-circle-2",
    color: "green",
    outputChannels: [
      { name: "approved", label: "Approved" },
      { name: "rejected", label: "Rejected" },
    ],
    configuration: [
      {
        name: "approvers",
        label: "Approvers",
        type: "string",
        required: true,
      },
    ],
  },
  {
    name: "filter",
    label: "Filter",
    description: "Filter items based on a condition",
    icon: "filter",
    color: "purple",
    outputChannels: [{ name: "default", label: "Next" }],
    configuration: [
      {
        name: "predicate",
        label: "Predicate",
        type: "string",
        required: true,
      },
    ],
  },
];
