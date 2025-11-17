import type { Meta, StoryObj } from "@storybook/react";
import { Button } from "./button";
import { MaterialSymbol } from "../MaterialSymbol/material-symbol";

const meta: Meta<typeof Button> = {
  title: "Components/Button",
  component: Button,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    color: {
      control: "select",
      options: [
        "dark/zinc",
        "light",
        "dark/white",
        "dark",
        "white",
        "zinc",
        "indigo",
        "cyan",
        "red",
        "orange",
        "amber",
        "yellow",
        "lime",
        "green",
        "emerald",
        "teal",
        "sky",
        "blue",
        "violet",
        "purple",
        "fuchsia",
        "pink",
        "rose",
      ],
    },
    outline: {
      control: "boolean",
    },
    plain: {
      control: "boolean",
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
    children: "Button",
  },
};

export const Colors: Story = {
  args: {
    children: "Button",
    color: "blue",
  },
};

export const Outline: Story = {
  args: {
    children: "Button",
    outline: true,
  },
};

export const Plain: Story = {
  args: {
    children: "Button",
    plain: true,
  },
};

export const Disabled: Story = {
  args: {
    children: "Button",
    disabled: true,
  },
};

export const WithIcon: Story = {
  args: {
    children: (
      <>
        <span data-slot="icon" className="material-symbols-outlined select-none" aria-hidden="true">
          add
        </span>
        Button with Icon
      </>
    ),
  },
};
