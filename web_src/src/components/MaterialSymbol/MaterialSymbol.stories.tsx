import type { Meta, StoryObj } from '@storybook/react'
import { MaterialSymbol, MaterialSymbolFilled, MaterialSymbolLight, MaterialSymbolBold } from './material-symbol'

const meta: Meta<typeof MaterialSymbol> = {
  title: 'Components/MaterialSymbol',
  component: MaterialSymbol,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    name: {
      control: 'text',
      description: 'Material Symbol name (e.g., home, settings, person)',
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg', 'xl'],
    },
    fill: {
      control: 'select',
      options: [0, 1],
      description: '0 = outlined, 1 = filled',
    },
    weight: {
      control: 'select',
      options: [100, 200, 300, 400, 500, 600, 700],
    },
    grade: {
      control: { type: 'range', min: -25, max: 200, step: 25 },
    },
    opticalSize: {
      control: { type: 'range', min: 20, max: 48, step: 4 },
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    name: 'home',
  },
}

export const Filled: Story = {
  args: {
    name: 'favorite',
    fill: 1,
  },
}

export const Sizes: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <MaterialSymbol name="star" size="sm" />
      <MaterialSymbol name="star" size="md" />
      <MaterialSymbol name="star" size="lg" />
      <MaterialSymbol name="star" size="xl" />
    </div>
  ),
}

export const Weights: Story = {
  render: () => (
    <div className="flex items-center gap-4">
      <MaterialSymbol name="settings" weight={100} />
      <MaterialSymbol name="settings" weight={300} />
      <MaterialSymbol name="settings" weight={400} />
      <MaterialSymbol name="settings" weight={600} />
      <MaterialSymbol name="settings" weight={700} />
    </div>
  ),
}

export const CommonIcons: Story = {
  render: () => (
    <div className="grid grid-cols-6 gap-4 items-center">
      <div className="text-center">
        <MaterialSymbol name="home" className="mb-1" />
        <div className="text-xs">home</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="settings" className="mb-1" />
        <div className="text-xs">settings</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="person" className="mb-1" />
        <div className="text-xs">person</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="favorite" className="mb-1" />
        <div className="text-xs">favorite</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="search" className="mb-1" />
        <div className="text-xs">search</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="menu" className="mb-1" />
        <div className="text-xs">menu</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="dashboard" className="mb-1" />
        <div className="text-xs">dashboard</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="notifications" className="mb-1" />
        <div className="text-xs">notifications</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="mail" className="mb-1" />
        <div className="text-xs">mail</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="folder" className="mb-1" />
        <div className="text-xs">folder</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="edit" className="mb-1" />
        <div className="text-xs">edit</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="delete" className="mb-1" />
        <div className="text-xs">delete</div>
      </div>
    </div>
  ),
}

export const NavigationIcons: Story = {
  render: () => (
    <div className="grid grid-cols-4 gap-4 items-center">
      <div className="text-center">
        <MaterialSymbol name="canvas" className="mb-1" />
        <div className="text-xs">canvas</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="key" className="mb-1" />
        <div className="text-xs">key</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="admin_panel_settings" className="mb-1" />
        <div className="text-xs">admin_panel_settings</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="bolt" className="mb-1" />
        <div className="text-xs">bolt</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="integration_instructions" className="mb-1" />
        <div className="text-xs">integration_instructions</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="workflow" className="mb-1" />
        <div className="text-xs">workflow</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="schema" className="mb-1" />
        <div className="text-xs">schema</div>
      </div>
      <div className="text-center">
        <MaterialSymbol name="hub" className="mb-1" />
        <div className="text-xs">hub</div>
      </div>
    </div>
  ),
}

export const PresetVariants: Story = {
  render: () => (
    <div className="flex items-center gap-6">
      <div className="text-center">
        <MaterialSymbol name="star" className="mb-1" />
        <div className="text-xs">Default</div>
      </div>
      <div className="text-center">
        <MaterialSymbolFilled name="star" className="mb-1" />
        <div className="text-xs">Filled</div>
      </div>
      <div className="text-center">
        <MaterialSymbolLight name="star" className="mb-1" />
        <div className="text-xs">Light</div>
      </div>
      <div className="text-center">
        <MaterialSymbolBold name="star" className="mb-1" />
        <div className="text-xs">Bold</div>
      </div>
    </div>
  ),
}

export const Interactive: Story = {
  render: () => (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold">Interactive Material Symbols</h3>
      <div className="flex gap-4">
        <button className="p-2 rounded hover:bg-gray-100 transition-colors">
          <MaterialSymbol name="thumb_up" className="text-blue-600" />
        </button>
        <button className="p-2 rounded hover:bg-gray-100 transition-colors">
          <MaterialSymbol name="share" className="text-green-600" />
        </button>
        <button className="p-2 rounded hover:bg-gray-100 transition-colors">
          <MaterialSymbol name="bookmark" className="text-yellow-600" />
        </button>
        <button className="p-2 rounded hover:bg-gray-100 transition-colors">
          <MaterialSymbol name="more_vert" className="text-gray-600" />
        </button>
      </div>
    </div>
  ),
}