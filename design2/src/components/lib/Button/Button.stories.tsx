import type { Meta, StoryObj } from '@storybook/react'
import { Button } from './button'

const meta: Meta<typeof Button> = {
  title: 'Components/Button',
  component: Button,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    color: {
      control: 'select',
      options: [
        'dark/zinc',
        'light',
        'dark/white',
        'dark',
        'white',
        'zinc',
        'indigo',
        'cyan',
        'red',
        'orange',
        'amber',
        'yellow',
        'lime',
        'green',
        'emerald',
        'teal',
        'sky',
        'blue',
        'violet',
        'purple',
        'fuchsia',
        'pink',
        'rose',
      ],
    },
    outline: {
      control: 'boolean',
    },
    plain: {
      control: 'boolean',
    },
    disabled: {
      control: 'boolean',
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    children: 'Button',
  },
}

export const Colors: Story = {
  render: () => (
    <div className="flex flex-wrap gap-4">
      <Button color="indigo">Indigo</Button>
      <Button color="blue">Blue</Button>
      <Button color="green">Green</Button>
      <Button color="red">Red</Button>
      <Button color="orange">Orange</Button>
      <Button color="amber">Amber</Button>
      <Button color="purple">Purple</Button>
      <Button color="pink">Pink</Button>
    </div>
  ),
}

export const Variants: Story = {
  render: () => (
    <div className="flex flex-col gap-4">
      <div className="flex gap-4">
        <Button>Solid</Button>
        <Button outline>Outline</Button>
        <Button plain>Plain</Button>
      </div>
      <div className="flex gap-4">
        <Button color="indigo">Solid Indigo</Button>
        <Button color="indigo" outline>Outline Indigo</Button>
        <Button color="indigo" plain>Plain Indigo</Button>
      </div>
    </div>
  ),
}

export const States: Story = {
  render: () => (
    <div className="flex gap-4">
      <Button>Normal</Button>
      <Button disabled>Disabled</Button>
    </div>
  ),
}

export const Sizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <Button className="text-sm px-3 py-1.5">Small</Button>
      <Button>Default</Button>
      <Button className="text-lg px-6 py-3">Large</Button>
    </div>
  ),
}

export const WithIcons: Story = {
  render: () => (
    <div className="flex gap-4">
      <Button>
        <svg className="size-4" data-slot="icon" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        Add Item
      </Button>
      <Button color="red">
        Delete
        <svg className="size-4" data-slot="icon" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
        </svg>
      </Button>
    </div>
  ),
}

export const Interactive: Story = {
  args: {
    children: 'Click me',
    onClick: () => alert('Button clicked!'),
  },
}

export const AsLink: Story = {
  args: {
    href: '#',
    children: 'Link Button',
    color: 'blue',
  },
}