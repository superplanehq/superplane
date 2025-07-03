import type { Meta, StoryObj } from '@storybook/react'
import { Avatar, AvatarButton } from './avatar'

const meta: Meta<typeof Avatar> = {
  title: 'Components/Avatar',
  component: Avatar,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    src: { control: 'text' },
    square: { control: 'boolean' },
    initials: { control: 'text' },
    alt: { control: 'text' },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    src: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80',
    alt: 'John Doe',
  },
}

export const WithInitials: Story = {
  args: {
    initials: 'JD',
    alt: 'John Doe',
  },
}

export const Square: Story = {
  args: {
    src: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80',
    square: true,
    alt: 'John Doe',
  },
}

export const SquareWithInitials: Story = {
  args: {
    initials: 'JD',
    square: true,
    alt: 'John Doe',
  },
}

export const Sizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <Avatar
        src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
        className="size-8"
        alt="Small"
      />
      <Avatar
        src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
        className="size-12"
        alt="Medium"
      />
      <Avatar
        src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
        className="size-16"
        alt="Large"
      />
      <Avatar
        src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
        className="size-20"
        alt="Extra Large"
      />
    </div>
  ),
}

export const InitialsSizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <Avatar initials="SM" className="size-8" alt="Small" />
      <Avatar initials="MD" className="size-12" alt="Medium" />
      <Avatar initials="LG" className="size-16" alt="Large" />
      <Avatar initials="XL" className="size-20" alt="Extra Large" />
    </div>
  ),
}

export const Group: Story = {
  render: () => (
    <div className="flex -space-x-2">
      <Avatar
        src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
        className="size-10 ring-2 ring-white"
        alt="Person 1"
      />
      <Avatar
        src="https://images.unsplash.com/photo-1517365830460-955ce3ccd263?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
        className="size-10 ring-2 ring-white"
        alt="Person 2"
      />
      <Avatar
        initials="AB"
        className="size-10 ring-2 ring-white"
        alt="Person 3"
      />
      <Avatar
        initials="+3"
        className="size-10 ring-2 ring-white bg-zinc-100 text-zinc-600"
        alt="3 more people"
      />
    </div>
  ),
}

export const AsButton: Story = {
  render: () => (
    <div className="flex gap-4">
      <AvatarButton
        src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
        alt="Click me"
        onClick={() => alert('Avatar clicked!')}
      />
      <AvatarButton
        initials="JD"
        alt="Click me"
        onClick={() => alert('Avatar with initials clicked!')}
      />
    </div>
  ),
}

export const AsLink: Story = {
  render: () => (
    <div className="flex gap-4">
      <AvatarButton
        href="#"
        src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?ixlib=rb-1.2.1&ixid=eyJhcHBfaWQiOjEyMDd9&auto=format&fit=facearea&facepad=2&w=256&h=256&q=80"
        alt="Profile link"
      />
      <AvatarButton
        href="#"
        initials="JD"
        alt="Profile link"
      />
    </div>
  ),
}