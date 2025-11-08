import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import { Select } from "./index";

const meta: Meta<typeof Select> = {
  title: "Components/Select",
  component: Select,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    error: {
      control: "boolean",
    },
    disabled: {
      control: "boolean",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const options = [
  { value: "option1", label: "Option 1" },
  { value: "option2", label: "Option 2" },
  { value: "option3", label: "Option 3" },
  { value: "option4", label: "Option 4" },
];

export const Default: Story = {
  render: () => {
    const [value, setValue] = useState("");
    return (
      <div style={{ width: "300px" }}>
        <Select options={options} value={value} onChange={setValue} placeholder="Select an option..." />
      </div>
    );
  },
};

export const WithValue: Story = {
  render: () => {
    const [value, setValue] = useState("option2");
    return (
      <div style={{ width: "300px" }}>
        <Select options={options} value={value} onChange={setValue} />
      </div>
    );
  },
};

export const Error: Story = {
  render: () => {
    const [value, setValue] = useState("");
    return (
      <div style={{ width: "300px" }}>
        <Select options={options} value={value} onChange={setValue} placeholder="Select an option..." error={true} />
      </div>
    );
  },
};

export const Disabled: Story = {
  render: () => {
    const [value, setValue] = useState("option2");
    return (
      <div style={{ width: "300px" }}>
        <Select options={options} value={value} onChange={setValue} disabled={true} />
      </div>
    );
  },
};

export const Interactive: Story = {
  render: () => {
    const [value, setValue] = useState("");
    const longOptions = [
      { value: "hourly", label: "Hourly" },
      { value: "daily", label: "Daily" },
      { value: "weekly", label: "Weekly" },
      { value: "monthly", label: "Monthly" },
      { value: "quarterly", label: "Quarterly" },
      { value: "yearly", label: "Yearly" },
    ];

    return (
      <div style={{ width: "300px" }}>
        <Select options={longOptions} value={value} onChange={setValue} placeholder="Select a frequency..." />
        <p style={{ marginTop: "10px", fontSize: "12px", color: "#666" }}>Selected: {value || "None"}</p>
        <p style={{ fontSize: "10px", color: "#666" }}>
          Try using keyboard navigation (Arrow keys, Enter, Space, Escape)
        </p>
      </div>
    );
  },
};

export const EmptyOptions: Story = {
  render: () => {
    const [value, setValue] = useState("");

    return (
      <div style={{ width: "300px" }}>
        <Select options={[]} value={value} onChange={setValue} placeholder="No options available..." />
      </div>
    );
  },
};
