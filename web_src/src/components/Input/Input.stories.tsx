import React from "react";
import type { Meta, StoryObj } from "@storybook/react";
import { Input, InputGroup } from "./input";

const meta: Meta<typeof Input> = {
  title: "Components/Input",
  component: Input,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    type: {
      control: "select",
      options: [
        "text",
        "email",
        "password",
        "number",
        "search",
        "tel",
        "url",
        "date",
        "datetime-local",
        "month",
        "time",
        "week",
      ],
    },
    disabled: {
      control: "boolean",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    placeholder: "Enter text...",
  },
};

export const Email: Story = {
  args: {
    type: "email",
    placeholder: "Enter email...",
  },
};

export const Password: Story = {
  args: {
    type: "password",
    placeholder: "Enter password...",
  },
};

export const Number: Story = {
  args: {
    type: "number",
    placeholder: "Enter number...",
  },
};

export const Disabled: Story = {
  args: {
    placeholder: "Disabled input",
    disabled: true,
  },
};

export const WithIcon: Story = {
  render: (args) => (
    <InputGroup>
      <span data-slot="icon" className="material-symbols-outlined select-none" aria-hidden="true">
        person
      </span>
      <Input {...args} />
    </InputGroup>
  ),
  args: {
    placeholder: "Search users...",
  },
};

export const WithRightIcon: Story = {
  render: (args) => (
    <InputGroup>
      <Input {...args} />
      <span data-slot="icon" className="material-symbols-outlined select-none" aria-hidden="true">
        search
      </span>
    </InputGroup>
  ),
  args: {
    placeholder: "Search...",
  },
};
