import React from 'react'
import type { Meta, StoryObj } from '@storybook/react'
import { Switch } from './switch'

const meta: Meta<typeof Switch> = {
  title: 'Components/Switch',
  component: Switch,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    checked: {
      control: 'boolean',
    },
    disabled: {
      control: 'boolean',
    },
    color: {
      control: 'select',
      options: ['blue', 'green', 'indigo'],
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {},
}

export const Checked: Story = {
  args: {
    checked: true,
  },
}

export const Disabled: Story = {
  args: {
    disabled: true,
  },
}

export const CheckedDisabled: Story = {
  args: {
    checked: true,
    disabled: true,
  },
}

export const Colors: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <Switch checked onChange={() => {}} color="blue" />
      <Switch checked onChange={() => {}} color="green" />
      <Switch checked onChange={() => {}} color="indigo" />
    </div>
  ),
}