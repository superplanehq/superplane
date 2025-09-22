import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { ValidationField } from './index'
import { Select } from '../Select'

const meta: Meta<typeof ValidationField> = {
  title: 'Components/ValidationField',
  component: ValidationField,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    required: {
      control: 'boolean',
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    label: 'Field Label',
    children: (
      <input
        type="text"
        placeholder="Enter value..."
        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
    ),
  },
}

export const Required: Story = {
  args: {
    label: 'Required Field',
    required: true,
    children: (
      <input
        type="text"
        placeholder="This field is required..."
        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
    ),
  },
}

export const WithError: Story = {
  args: {
    label: 'Field with Error',
    required: true,
    error: 'This field is required and cannot be empty',
    children: (
      <input
        type="text"
        placeholder="Enter value..."
        className="w-full px-3 py-2 border border-red-500 dark:border-red-400 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-red-500"
      />
    ),
  },
}

export const WithSelect: Story = {
  render: () => {
    const [value, setValue] = useState('')
    const options = [
      { value: 'option1', label: 'Option 1' },
      { value: 'option2', label: 'Option 2' },
      { value: 'option3', label: 'Option 3' },
    ]

    return (
      <ValidationField label="Select Field">
        <Select
          options={options}
          value={value}
          onChange={setValue}
          placeholder="Select an option..."
        />
      </ValidationField>
    )
  },
}

export const WithTextarea: Story = {
  args: {
    label: 'Message',
    children: (
      <textarea
        rows={4}
        placeholder="Enter your message..."
        className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
      />
    ),
  },
}