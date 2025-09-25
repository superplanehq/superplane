import React from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { BrowserRouter } from 'react-router-dom'
import { CanvasCard } from './canvas-card'
import { mockCanvas, shortCanvas, noDescriptionCanvas } from '../../../test/__mocks__/canvas'

const meta: Meta<typeof CanvasCard> = {
  title: 'Components/CanvasCard',
  component: CanvasCard,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['grid', 'list'],
    },
  },
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="bg-zinc-50 dark:bg-zinc-900 p-4">
          <Story />
        </div>
      </BrowserRouter>
    ),
  ],
}

export default meta
type Story = StoryObj<typeof meta>

export const ListView: Story = {
  args: {
    canvas: mockCanvas,
    organizationId: 'org-123',
    variant: 'list',
  },
}

export const GridViewShortContent: Story = {
  args: {
    canvas: shortCanvas,
    organizationId: 'org-123',
    variant: 'grid',
  },
}

export const ListLayout: Story = {
  render: () => (
    <div className="space-y-2 max-w-4xl">
      <CanvasCard canvas={mockCanvas} organizationId="org-123" variant="list" />
      <CanvasCard canvas={shortCanvas} organizationId="org-123" variant="list" />
      <CanvasCard canvas={noDescriptionCanvas} organizationId="org-123" variant="list" />
      <CanvasCard canvas={{...mockCanvas, id: '4', name: 'Another Canvas'}} organizationId="org-123" variant="list" />
    </div>
  ),
}