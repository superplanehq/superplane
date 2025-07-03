import type { Meta, StoryObj } from '@storybook/react'
import { Badge, BadgeButton } from './badge'

const meta: Meta<typeof Badge> = {
  title: 'Components/Badge',
  component: Badge,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    color: {
      control: 'select',
      options: [
        'red',
        'orange',
        'amber',
        'yellow',
        'lime',
        'green',
        'emerald',
        'teal',
        'cyan',
        'sky',
        'blue',
        'indigo',
        'violet',
        'purple',
        'fuchsia',
        'pink',
        'rose',
        'zinc',
      ],
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    children: 'Badge',
  },
}

export const Colors: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Badge color="red">Red</Badge>
      <Badge color="orange">Orange</Badge>
      <Badge color="amber">Amber</Badge>
      <Badge color="yellow">Yellow</Badge>
      <Badge color="lime">Lime</Badge>
      <Badge color="green">Green</Badge>
      <Badge color="emerald">Emerald</Badge>
      <Badge color="teal">Teal</Badge>
      <Badge color="cyan">Cyan</Badge>
      <Badge color="sky">Sky</Badge>
      <Badge color="blue">Blue</Badge>
      <Badge color="indigo">Indigo</Badge>
      <Badge color="violet">Violet</Badge>
      <Badge color="purple">Purple</Badge>
      <Badge color="fuchsia">Fuchsia</Badge>
      <Badge color="pink">Pink</Badge>
      <Badge color="rose">Rose</Badge>
      <Badge color="zinc">Zinc</Badge>
    </div>
  ),
}

export const WithIcons: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Badge color="green">
        <svg className="size-3" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
            clipRule="evenodd"
          />
        </svg>
        Completed
      </Badge>
      <Badge color="yellow">
        <svg className="size-3" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
            clipRule="evenodd"
          />
        </svg>
        Warning
      </Badge>
      <Badge color="red">
        <svg className="size-3" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
            clipRule="evenodd"
          />
        </svg>
        Error
      </Badge>
    </div>
  ),
}

export const Status: Story = {
  render: () => (
    <div className="space-y-4">
      <div className="flex gap-2">
        <Badge color="green">Active</Badge>
        <Badge color="yellow">Pending</Badge>
        <Badge color="red">Inactive</Badge>
        <Badge color="zinc">Draft</Badge>
      </div>
      <div className="flex gap-2">
        <Badge color="blue">New</Badge>
        <Badge color="purple">Premium</Badge>
        <Badge color="orange">Hot</Badge>
        <Badge color="pink">Popular</Badge>
      </div>
    </div>
  ),
}

export const WithDot: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Badge color="green">
        <svg className="size-1.5 fill-current" viewBox="0 0 6 6">
          <circle cx={3} cy={3} r={3} />
        </svg>
        Online
      </Badge>
      <Badge color="yellow">
        <svg className="size-1.5 fill-current" viewBox="0 0 6 6">
          <circle cx={3} cy={3} r={3} />
        </svg>
        Away
      </Badge>
      <Badge color="red">
        <svg className="size-1.5 fill-current" viewBox="0 0 6 6">
          <circle cx={3} cy={3} r={3} />
        </svg>
        Offline
      </Badge>
    </div>
  ),
}

export const AsButton: Story = {
  render: () => (
    <div className="flex gap-2">
      <BadgeButton color="blue" onClick={() => alert('Badge clicked!')}>
        Clickable
      </BadgeButton>
      <BadgeButton color="green" onClick={() => alert('Status changed!')}>
        Toggle Status
      </BadgeButton>
    </div>
  ),
}

export const AsLink: Story = {
  render: () => (
    <div className="flex gap-2">
      <BadgeButton href="#" color="blue">
        Link Badge
      </BadgeButton>
      <BadgeButton href="#" color="purple">
        View Details
      </BadgeButton>
    </div>
  ),
}

export const Numbers: Story = {
  render: () => (
    <div className="flex gap-2">
      <Badge color="red">3</Badge>
      <Badge color="blue">12</Badge>
      <Badge color="green">99+</Badge>
      <Badge color="zinc">1,234</Badge>
    </div>
  ),
}