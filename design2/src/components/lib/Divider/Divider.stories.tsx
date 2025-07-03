import type { Meta, StoryObj } from '@storybook/react'
import { Divider } from './divider'

const meta: Meta<typeof Divider> = {
  title: 'Components/Divider',
  component: Divider,
  parameters: {
    layout: 'padded',
  },
  tags: ['autodocs'],
  argTypes: {
    soft: { control: 'boolean' },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {},
}

export const Soft: Story = {
  args: {
    soft: true,
  },
}

export const InContent: Story = {
  render: () => (
    <div className="max-w-md space-y-4">
      <div>
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white">Section 1</h3>
        <p className="text-zinc-600 dark:text-zinc-400">
          This is the first section with some content to demonstrate how dividers work in context.
        </p>
      </div>
      
      <Divider />
      
      <div>
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white">Section 2</h3>
        <p className="text-zinc-600 dark:text-zinc-400">
          This is the second section showing how dividers separate content visually.
        </p>
      </div>
      
      <Divider soft />
      
      <div>
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white">Section 3</h3>
        <p className="text-zinc-600 dark:text-zinc-400">
          This section is separated by a soft divider, which has a more subtle appearance.
        </p>
      </div>
    </div>
  ),
}

export const InCard: Story = {
  render: () => (
    <div className="max-w-sm bg-white dark:bg-zinc-900 rounded-lg shadow-lg border border-zinc-200 dark:border-zinc-700">
      <div className="p-6">
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white">Card Header</h3>
        <p className="text-sm text-zinc-600 dark:text-zinc-400">Some description text here</p>
      </div>
      
      <Divider />
      
      <div className="p-6">
        <h4 className="font-medium text-zinc-900 dark:text-white">Card Body</h4>
        <p className="text-sm text-zinc-600 dark:text-zinc-400">Main content goes here</p>
      </div>
      
      <Divider soft />
      
      <div className="p-6">
        <div className="flex gap-2">
          <button className="px-3 py-1 bg-blue-500 text-white rounded text-sm">Action</button>
          <button className="px-3 py-1 bg-zinc-200 text-zinc-700 rounded text-sm">Cancel</button>
        </div>
      </div>
    </div>
  ),
}

export const InList: Story = {
  render: () => (
    <div className="max-w-md bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-700">
      <div className="p-4">
        <h3 className="font-semibold text-zinc-900 dark:text-white mb-4">Team Members</h3>
        
        <div className="space-y-0">
          <div className="flex items-center gap-3 py-3">
            <div className="w-8 h-8 bg-blue-500 rounded-full"></div>
            <div>
              <div className="font-medium text-zinc-900 dark:text-white">John Doe</div>
              <div className="text-sm text-zinc-600 dark:text-zinc-400">john@example.com</div>
            </div>
          </div>
          
          <Divider soft />
          
          <div className="flex items-center gap-3 py-3">
            <div className="w-8 h-8 bg-green-500 rounded-full"></div>
            <div>
              <div className="font-medium text-zinc-900 dark:text-white">Jane Smith</div>
              <div className="text-sm text-zinc-600 dark:text-zinc-400">jane@example.com</div>
            </div>
          </div>
          
          <Divider soft />
          
          <div className="flex items-center gap-3 py-3">
            <div className="w-8 h-8 bg-purple-500 rounded-full"></div>
            <div>
              <div className="font-medium text-zinc-900 dark:text-white">Bob Johnson</div>
              <div className="text-sm text-zinc-600 dark:text-zinc-400">bob@example.com</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  ),
}

export const Comparison: Story = {
  render: () => (
    <div className="space-y-8">
      <div>
        <h3 className="text-lg font-semibold mb-4 text-zinc-900 dark:text-white">Regular Divider</h3>
        <div className="space-y-4">
          <p className="text-zinc-600 dark:text-zinc-400">Content above the divider</p>
          <Divider />
          <p className="text-zinc-600 dark:text-zinc-400">Content below the divider</p>
        </div>
      </div>
      
      <div>
        <h3 className="text-lg font-semibold mb-4 text-zinc-900 dark:text-white">Soft Divider</h3>
        <div className="space-y-4">
          <p className="text-zinc-600 dark:text-zinc-400">Content above the soft divider</p>
          <Divider soft />
          <p className="text-zinc-600 dark:text-zinc-400">Content below the soft divider</p>
        </div>
      </div>
    </div>
  ),
}