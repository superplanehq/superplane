import type { Meta, StoryObj } from '@storybook/react';

import { Checkbox } from '../checkbox';
import { Label } from './index';

const meta: Meta<typeof Label> = {
  title: 'shadcn Primitives/Label',
  component: Label,
  tags: ['autodocs'],
  argTypes: {
    children: {
      control: { type: 'text' },
    },
  },
  parameters: {
    layout: 'centered',
  },
};

export default meta;
type Story = StoryObj<typeof Label>;

export const Default: Story = {
  args: {
    htmlFor: 'email',
    children: 'Your email address',
  },
};

export const WithCheckbox: Story = {
  render: (args) => (
    <div className="flex items-center space-x-2">
      <Checkbox id="terms" />
      <Label {...args} htmlFor="terms" />
    </div>
  ),
  args: {
    children: 'Accept terms and conditions',
  },
};
