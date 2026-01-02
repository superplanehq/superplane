import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { AutoCompleteInput } from "./AutoCompleteInput";

const meta: Meta<typeof AutoCompleteInput> = {
  title: "Components/AutoCompleteInput",
  component: AutoCompleteInput,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    disabled: {
      control: "boolean",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Example objects for different scenarios
const simpleObject = {
  name: "John",
  age: 30,
  city: "New York",
};

const nestedObject = {
  user: {
    name: "John",
    profile: {
      email: "john@example.com",
      settings: {
        theme: "dark",
        notifications: true,
      },
    },
    address: {
      street: "123 Main St",
      city: "New York",
      zipCode: "10001",
    },
  },
};

const arrayObject = {
  users: [
    { id: 1, name: "Alice", role: "admin" },
    { id: 2, name: "Bob", role: "user" },
  ],
  products: [
    { id: "p1", title: "Laptop", price: 999 },
    { id: "p2", title: "Mouse", price: 25 },
  ],
};

const complexObject = {
  company: {
    name: "TechCorp",
    departments: [
      {
        name: "Engineering",
        employees: [
          {
            id: 1,
            name: "Alice",
            skills: ["React", "TypeScript", "Node.js"],
            contact: {
              email: "alice@techcorp.com",
              phone: "555-0123",
            },
          },
          {
            id: 2,
            name: "Bob",
            skills: ["Python", "Django", "PostgreSQL"],
            contact: {
              email: "bob@techcorp.com",
              phone: "555-0124",
            },
          },
        ],
      },
      {
        name: "Marketing",
        employees: [
          {
            id: 3,
            name: "Carol",
            skills: ["SEO", "Content Strategy", "Analytics"],
            contact: {
              email: "carol@techcorp.com",
              phone: "555-0125",
            },
          },
        ],
      },
    ],
    metadata: {
      founded: 2010,
      location: "San Francisco",
      revenue: 50000000,
    },
  },
};

export const Simple: Story = {
  args: {
    exampleObj: simpleObject,
    placeholder: "Type to autocomplete simple object paths...",
  },
};

export const NestedObject: Story = {
  args: {
    exampleObj: nestedObject,
    placeholder: "Type to autocomplete nested object paths...",
  },
};

export const WithArrays: Story = {
  args: {
    exampleObj: arrayObject,
    placeholder: "Type to autocomplete array paths...",
  },
};

export const Complex: Story = {
  args: {
    exampleObj: complexObject,
    placeholder: "Type to autocomplete complex nested paths...",
  },
};

export const Controlled: Story = {
  render: (args) => {
    const [value, setValue] = React.useState("user");

    return (
      <div className="space-y-4">
        <AutoCompleteInput {...args} value={value} onChange={setValue} />
        <div className="text-sm text-gray-500">
          Current value: <code className="bg-gray-100 px-1 py-0.5 rounded">{value}</code>
        </div>
      </div>
    );
  },
  args: {
    exampleObj: nestedObject,
    placeholder: "Controlled input example...",
  },
};

export const Disabled: Story = {
  args: {
    exampleObj: simpleObject,
    placeholder: "Disabled autocomplete input",
    disabled: true,
    value: "name",
  },
};

export const WithInitialValue: Story = {
  args: {
    exampleObj: complexObject,
    placeholder: "Input with initial value...",
    value: "company.departments[0]",
  },
};
