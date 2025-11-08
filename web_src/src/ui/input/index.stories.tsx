import type { Meta, StoryObj } from "@storybook/react";

import { Button } from "../button";
import { Input } from "./index";
import { Label } from "../label";

const meta: Meta<typeof Input> = {
  title: "shadcn Primitives/Input",
  component: Input,
  tags: ["autodocs"],
  argTypes: {
    type: {
      control: { type: "select" },
      options: ["text", "password", "email", "number", "search", "tel", "url", "file"],
    },
    disabled: {
      control: { type: "boolean" },
    },
  },
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof Input>;

export const Default: Story = {
  args: {
    type: "text",
    placeholder: "Enter text...",
    disabled: false,
  },
};

export const WithLabel: Story = {
  render: (args) => (
    <div className="grid w-full max-w-sm items-center gap-1.5">
      <Label htmlFor="email-2">Email</Label>
      <Input {...args} type="email" id="email-2" placeholder="Email" />
    </div>
  ),
  args: {
    disabled: false,
  },
};

export const WithButton: Story = {
  render: (args) => (
    <div className="flex w-full max-w-sm items-center space-x-2">
      <Input {...args} type="email" placeholder="Email" />
      <Button type="submit">Subscribe</Button>
    </div>
  ),
  args: {
    disabled: false,
  },
};

export const File: Story = {
  render: (args) => (
    <div className="grid w-full max-w-sm items-center gap-1.5">
      <Label htmlFor="picture">Picture</Label>
      <Input {...args} id="picture" type="file" />
    </div>
  ),
  args: {
    disabled: false,
  },
};
