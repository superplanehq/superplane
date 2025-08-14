import type { Meta, StoryObj } from '@storybook/react';
import { MaterialSymbol } from '../material-symbol';

const meta: Meta<typeof MaterialSymbol> = {
  title: 'Components/MaterialSymbol',
  component: MaterialSymbol,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: 'Material Symbol icons with support for various sizes, weights, and fills. Supports both named sizes (sm, md, lg, etc.) and exact pixel sizes (32, 36, 40, 48, 56, 60, 64).'
      }
    }
  },
  argTypes: {
    name: {
      control: 'text',
      description: 'The Material Symbol name'
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg', 'xl', '2xl', '3xl', '4xl', '5xl', '6xl', '7xl', 32, 36, 40, 48, 56, 60, 64]
    },
    fill: {
      control: 'select',
      options: [0, 1],
      description: '0 = outlined, 1 = filled'
    },
    weight: {
      control: 'select',
      options: [100, 200, 300, 400, 500, 600, 700]
    },
    grade: {
      control: { type: 'range', min: -25, max: 200, step: 25 }
    },
    opticalSize: {
      control: { type: 'range', min: 20, max: 48, step: 4 }
    }
  },
  decorators: [
    (Story) => (
      <div className="p-8 bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof MaterialSymbol>;

// Default story
export const Default: Story = {
  args: {
    name: 'home',
    size: 'md'
  }
};

// Named sizes showcase
export const NamedSizes: Story = {
  render: () => (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold mb-4">Named Sizes</h3>
      <div className="grid grid-cols-2 gap-4 items-center">
        <div className="flex items-center gap-3">
          <MaterialSymbol name="star" size="sm" />
          <span className="text-sm">sm (14px)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="star" size="md" />
          <span className="text-sm">md (16px)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="star" size="lg" />
          <span className="text-sm">lg (20px)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="star" size="xl" />
          <span className="text-sm">xl (24px)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="star" size="2xl" />
          <span className="text-sm">2xl (24px)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="star" size="3xl" />
          <span className="text-sm">3xl (30px)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="star" size="4xl" />
          <span className="text-sm">4xl (36px)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="star" size="5xl" />
          <span className="text-sm">5xl (48px)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="star" size="6xl" />
          <span className="text-sm">6xl (60px)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="star" size="7xl" />
          <span className="text-sm">7xl (72px)</span>
        </div>
      </div>
    </div>
  )
};

// Pixel sizes showcase
export const PixelSizes: Story = {
  render: () => (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold mb-4">Exact Pixel Sizes</h3>
      <div className="grid grid-cols-2 gap-6 items-center">
        <div className="flex items-center gap-3">
          <MaterialSymbol name="favorite" size={32} />
          <span className="text-sm">32px</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="favorite" size={36} />
          <span className="text-sm">36px</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="favorite" size={40} />
          <span className="text-sm">40px</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="favorite" size={48} />
          <span className="text-sm">48px</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="favorite" size={56} />
          <span className="text-sm">56px</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="favorite" size={60} />
          <span className="text-sm">60px</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="favorite" size={64} />
          <span className="text-sm">64px</span>
        </div>
      </div>
    </div>
  )
};

// Fill variants
export const FillVariants: Story = {
  render: () => (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold mb-4">Fill Variants</h3>
      <div className="flex items-center gap-6">
        <div className="flex items-center gap-3">
          <MaterialSymbol name="favorite" size={48} fill={0} />
          <span className="text-sm">Outlined (fill=0)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="favorite" size={48} fill={1} />
          <span className="text-sm">Filled (fill=1)</span>
        </div>
      </div>
    </div>
  )
};

// Weight variants
export const WeightVariants: Story = {
  render: () => (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold mb-4">Weight Variants</h3>
      <div className="grid grid-cols-2 gap-4">
        <div className="flex items-center gap-3">
          <MaterialSymbol name="settings" size={40} weight={100} />
          <span className="text-sm">Weight 100</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="settings" size={40} weight={300} />
          <span className="text-sm">Weight 300</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="settings" size={40} weight={400} />
          <span className="text-sm">Weight 400 (default)</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="settings" size={40} weight={600} />
          <span className="text-sm">Weight 600</span>
        </div>
        <div className="flex items-center gap-3">
          <MaterialSymbol name="settings" size={40} weight={700} />
          <span className="text-sm">Weight 700</span>
        </div>
      </div>
    </div>
  )
};

// Large showcase
export const LargeShowcase: Story = {
  render: () => (
    <div className="space-y-6">
      <h3 className="text-lg font-semibold mb-4">Large Icon Showcase</h3>
      <div className="flex items-center justify-center gap-8 p-8 bg-zinc-50 dark:bg-zinc-800 rounded-lg">
        <MaterialSymbol name="cloud_sync" size={64} className="text-blue-500" />
        <MaterialSymbol name="notifications" size={56} fill={1} className="text-orange-500" />
        <MaterialSymbol name="favorite" size={48} fill={1} className="text-red-500" />
        <MaterialSymbol name="star" size={40} fill={1} className="text-yellow-500" />
      </div>
    </div>
  )
};