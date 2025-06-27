import type { Meta, StoryObj } from '@storybook/react'
import { Input, InputGroup } from './input'
import { Label, Field, Description } from '@headlessui/react'

const meta: Meta<typeof Input> = {
  title: 'Components/Input',
  component: Input,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    type: {
      control: 'select',
      options: ['text', 'email', 'password', 'number', 'tel', 'url', 'search', 'date', 'datetime-local', 'month', 'time', 'week'],
    },
    disabled: { control: 'boolean' },
    invalid: { control: 'boolean' },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    placeholder: 'Enter text...',
  },
}

export const Types: Story = {
  render: () => (
    <div className="space-y-4 w-64">
      <Field>
        <Label>Text Input</Label>
        <Input type="text" placeholder="Enter text" />
      </Field>
      <Field>
        <Label>Email Input</Label>
        <Input type="email" placeholder="Enter email" />
      </Field>
      <Field>
        <Label>Password Input</Label>
        <Input type="password" placeholder="Enter password" />
      </Field>
      <Field>
        <Label>Number Input</Label>
        <Input type="number" placeholder="Enter number" />
      </Field>
      <Field>
        <Label>URL Input</Label>
        <Input type="url" placeholder="https://example.com" />
      </Field>
      <Field>
        <Label>Search Input</Label>
        <Input type="search" placeholder="Search..." />
      </Field>
    </div>
  ),
}

export const DateInputs: Story = {
  render: () => (
    <div className="space-y-4 w-64">
      <Field>
        <Label>Date</Label>
        <Input type="date" />
      </Field>
      <Field>
        <Label>Date and Time</Label>
        <Input type="datetime-local" />
      </Field>
      <Field>
        <Label>Month</Label>
        <Input type="month" />
      </Field>
      <Field>
        <Label>Time</Label>
        <Input type="time" />
      </Field>
      <Field>
        <Label>Week</Label>
        <Input type="week" />
      </Field>
    </div>
  ),
}

export const States: Story = {
  render: () => (
    <div className="space-y-4 w-64">
      <Field>
        <Label>Normal</Label>
        <Input placeholder="Normal input" />
      </Field>
      <Field>
        <Label>Disabled</Label>
        <Input placeholder="Disabled input" disabled />
      </Field>
      <Field>
        <Label>Invalid</Label>
        <Input placeholder="Invalid input" invalid />
        <Description className="text-red-500 text-sm">This field has an error</Description>
      </Field>
      <Field>
        <Label>With Value</Label>
        <Input defaultValue="Pre-filled value" />
      </Field>
    </div>
  ),
}

export const WithIcons: Story = {
  render: () => (
    <div className="space-y-4 w-64">
      <Field>
        <Label>Search with Icon</Label>
        <InputGroup>
          <svg data-slot="icon" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
          </svg>
          <Input type="search" placeholder="Search..." />
        </InputGroup>
      </Field>
      
      <Field>
        <Label>Email with Icon</Label>
        <InputGroup>
          <svg data-slot="icon" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 12a4.5 4.5 0 1 1-9 0 4.5 4.5 0 0 1 9 0Zm0 0c0 1.657 1.007 3 2.25 3S21 13.657 21 12a9 9 0 1 0-2.636 6.364M16.5 12V8.25" />
          </svg>
          <Input type="email" placeholder="Enter your email" />
        </InputGroup>
      </Field>
      
      <Field>
        <Label>URL with Trailing Icon</Label>
        <InputGroup>
          <Input type="url" placeholder="https://example.com" />
          <svg data-slot="icon" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M13.19 8.688a4.5 4.5 0 0 1 1.242 7.244l-4.5 4.5a4.5 4.5 0 0 1-6.364-6.364l1.757-1.757m13.35-.622 1.757-1.757a4.5 4.5 0 0 0-6.364-6.364l-4.5 4.5a4.5 4.5 0 0 0 1.242 7.244" />
          </svg>
        </InputGroup>
      </Field>
    </div>
  ),
}

export const WithDescriptions: Story = {
  render: () => (
    <div className="space-y-6 w-80">
      <Field>
        <Label>Username</Label>
        <Input placeholder="Enter username" />
        <Description>Choose a unique username between 3-20 characters.</Description>
      </Field>
      
      <Field>
        <Label>Password</Label>
        <Input type="password" placeholder="Enter password" />
        <Description>Must be at least 8 characters with numbers and special characters.</Description>
      </Field>
      
      <Field>
        <Label>Website URL</Label>
        <InputGroup>
          <Input type="url" placeholder="https://example.com" />
          <svg data-slot="icon" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M13.19 8.688a4.5 4.5 0 0 1 1.242 7.244l-4.5 4.5a4.5 4.5 0 0 1-6.364-6.364l1.757-1.757m13.35-.622 1.757-1.757a4.5 4.5 0 0 0-6.364-6.364l-4.5 4.5a4.5 4.5 0 0 0 1.242 7.244" />
          </svg>
        </InputGroup>
        <Description>Enter the full URL including https://</Description>
      </Field>
    </div>
  ),
}

export const FormExample: Story = {
  render: () => (
    <form className="space-y-6 w-96">
      <div>
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-4">Contact Information</h3>
        
        <div className="space-y-4">
          <Field>
            <Label>Full Name *</Label>
            <Input type="text" placeholder="John Doe" required />
          </Field>
          
          <Field>
            <Label>Email Address *</Label>
            <InputGroup>
              <svg data-slot="icon" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 12a4.5 4.5 0 1 1-9 0 4.5 4.5 0 0 1 9 0Zm0 0c0 1.657 1.007 3 2.25 3S21 13.657 21 12a9 9 0 1 0-2.636 6.364M16.5 12V8.25" />
              </svg>
              <Input type="email" placeholder="john@example.com" required />
            </InputGroup>
          </Field>
          
          <Field>
            <Label>Phone Number</Label>
            <Input type="tel" placeholder="+1 (555) 123-4567" />
            <Description>Optional: Include country code</Description>
          </Field>
          
          <Field>
            <Label>Company Website</Label>
            <InputGroup>
              <Input type="url" placeholder="https://company.com" />
              <svg data-slot="icon" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" d="M13.19 8.688a4.5 4.5 0 0 1 1.242 7.244l-4.5 4.5a4.5 4.5 0 0 1-6.364-6.364l1.757-1.757m13.35-.622 1.757-1.757a4.5 4.5 0 0 0-6.364-6.364l-4.5 4.5a4.5 4.5 0 0 0 1.242 7.244" />
              </svg>
            </InputGroup>
          </Field>
        </div>
      </div>
      
      <div className="flex gap-2">
        <button type="submit" className="px-4 py-2 bg-blue-500 text-white rounded-md">
          Submit
        </button>
        <button type="button" className="px-4 py-2 bg-zinc-200 text-zinc-700 rounded-md">
          Cancel
        </button>
      </div>
    </form>
  ),
}