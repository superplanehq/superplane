import type { Meta, StoryObj } from '@storybook/react';
import { ComponentLibraryTooltip } from './index';

const meta: Meta<typeof ComponentLibraryTooltip> = {
  title: 'ui/ComponentLibraryTooltip',
  component: ComponentLibraryTooltip,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof meta>;

const sampleTabs = [
  {
    name: 'Blocks',
    subCategories: [
      {
        name: 'Primitives',
        options: [
          { name: 'Approval', blockColor: '#3B82F6', onClick: () => console.log('Approval clicked') },
          { name: 'Filter', blockColor: '#10B981', onClick: () => console.log('Filter clicked') },
          { name: 'HTTP', blockColor: '#F59E0B', onClick: () => console.log('HTTP clicked') },
          { name: 'If', blockColor: '#8B5CF6', onClick: () => console.log('If clicked') },
          { name: 'No operation', blockColor: '#6B7280', onClick: () => console.log('No operation clicked') },
          { name: 'Switch', blockColor: '#EF4444', onClick: () => console.log('Switch clicked') },
          { name: 'Wait', blockColor: '#06B6D4', onClick: () => console.log('Wait clicked') },
        ],
      },
      {
        name: 'Integrations',
        options: [
          {
            name: 'Argo CD',
            blockColor: '#FF6B35',
            onClick: () => console.log('Argo CD clicked'),
            subOptions: [
              { name: 'Listen to Something', blockColor: '#FF6B35', onClick: () => console.log('Argo CD - Listen to Something') },
              { name: 'Something Else', blockColor: '#FF6B35', onClick: () => console.log('Argo CD - Something Else') },
              { name: 'Other thing', blockColor: '#FF6B35', onClick: () => console.log('Argo CD - Other thing') },
            ],
          },
          {
            name: 'GitHub',
            blockColor: '#24292E',
            onClick: () => console.log('GitHub clicked'),
            subOptions: [
              { name: 'Listen to Something', blockColor: '#24292E', onClick: () => console.log('GitHub - Listen to Something') },
              { name: 'Something Else', blockColor: '#24292E', onClick: () => console.log('GitHub - Something Else') },
              { name: 'Other thing', blockColor: '#24292E', onClick: () => console.log('GitHub - Other thing') },
            ],
          },
          {
            name: 'DataDog',
            blockColor: '#632CA6',
            onClick: () => console.log('DataDog clicked'),
            subOptions: [
              { name: 'Listen to Something', blockColor: '#632CA6', onClick: () => console.log('DataDog - Listen to Something') },
              { name: 'Something Else', blockColor: '#632CA6', onClick: () => console.log('DataDog - Something Else') },
              { name: 'Other thing', blockColor: '#632CA6', onClick: () => console.log('DataDog - Other thing') },
            ],
          },
          {
            name: 'Kubernetes',
            blockColor: '#326CE5',
            onClick: () => console.log('Kubernetes clicked'),
            subOptions: [
              { name: 'Listen to Something', blockColor: '#326CE5', onClick: () => console.log('Kubernetes - Listen to Something') },
              { name: 'Something Else', blockColor: '#326CE5', onClick: () => console.log('Kubernetes - Something Else') },
              { name: 'Other thing', blockColor: '#326CE5', onClick: () => console.log('Kubernetes - Other thing') },
            ],
          },
          {
            name: 'PagerDuty',
            blockColor: '#06AC38',
            onClick: () => console.log('PagerDuty clicked'),
            subOptions: [
              { name: 'Listen to Something', blockColor: '#06AC38', onClick: () => console.log('PagerDuty - Listen to Something') },
              { name: 'Something Else', blockColor: '#06AC38', onClick: () => console.log('PagerDuty - Something Else') },
              { name: 'Other thing', blockColor: '#06AC38', onClick: () => console.log('PagerDuty - Other thing') },
            ],
          },
          {
            name: 'Snyk',
            blockColor: '#4C4A73',
            onClick: () => console.log('Snyk clicked'),
            subOptions: [
              { name: 'Listen to Something', blockColor: '#4C4A73', onClick: () => console.log('Snyk - Listen to Something') },
              { name: 'Something Else', blockColor: '#4C4A73', onClick: () => console.log('Snyk - Something Else') },
              { name: 'Other thing', blockColor: '#4C4A73', onClick: () => console.log('Snyk - Other thing') },
            ],
          },
          {
            name: 'Terraform',
            blockColor: '#7B42BC',
            onClick: () => console.log('Terraform clicked'),
            subOptions: [
              { name: 'Listen to Something', blockColor: '#7B42BC', onClick: () => console.log('Terraform - Listen to Something') },
              { name: 'Something Else', blockColor: '#7B42BC', onClick: () => console.log('Terraform - Something Else') },
              { name: 'Other thing', blockColor: '#7B42BC', onClick: () => console.log('Terraform - Other thing') },
            ],
          },
        ],
      },
    ],
  },
  {
    name: 'Groups',
    subCategories: [],
  },
];

export const Default: Story = {
  render: () => (
    <div className="p-8">
      <p className="mb-4 text-sm text-gray-600">Hover over the button below to see the tooltip:</p>
      <ComponentLibraryTooltip tabs={sampleTabs}>
        <button className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">
          Hover me
        </button>
      </ComponentLibraryTooltip>
    </div>
  ),
};

export const WithText: Story = {
  render: () => (
    <div className="p-8">
      <p className="mb-4 text-sm text-gray-600">Tooltip can wrap any element:</p>
      <ComponentLibraryTooltip tabs={sampleTabs}>
        <span className="text-blue-600 cursor-pointer underline">
          Hover this text
        </span>
      </ComponentLibraryTooltip>
    </div>
  ),
};

export const AlwaysOpen: Story = {
  render: () => (
    <div className="p-8">
      <p className="mb-4 text-sm text-gray-600">Tooltip always visible for development:</p>
      <div className="relative">
        <button className="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600">
          Always visible tooltip
        </button>
        <div className="absolute top-0 left-1/2 transform -translate-x-1/2 -translate-y-full mb-2 z-10">
          <div className="bg-white border-2 border-gray-200 rounded-md max-w-[700px] shadow-lg">
            <div className="flex border-b border-gray-200">
              <button className="px-4 py-2 text-sm font-medium text-blue-600 border-b-2 border-blue-600">
                Blocks
              </button>
              <button className="px-4 py-2 text-sm font-medium text-gray-600 hover:text-gray-800">
                Groups
              </button>
            </div>
            <div className="p-4">
              <div className="mb-4">
                <h3 className="font-medium text-gray-800 mb-2">Primitives</h3>
                <div className="space-y-1">
                  <div className="flex items-center justify-between w-full text-left px-3 py-2 text-sm text-gray-700 bg-gray-100 rounded">
                    <div className="flex items-center">
                      <div className="w-3 h-3 rounded mr-2" style={{ backgroundColor: '#3B82F6' }}></div>
                      Approval
                    </div>
                  </div>
                  <div className="flex items-center justify-between w-full text-left px-3 py-2 text-sm text-gray-700 rounded">
                    <div className="flex items-center">
                      <div className="w-3 h-3 rounded mr-2" style={{ backgroundColor: '#10B981' }}></div>
                      Filter
                    </div>
                  </div>
                </div>
              </div>
              <div className="mb-4">
                <h3 className="font-medium text-gray-800 mb-2">Integrations</h3>
                <div className="space-y-1">
                  <div className="flex items-center justify-between w-full text-left px-3 py-2 text-sm text-gray-700 rounded">
                    <div className="flex items-center">
                      <div className="w-3 h-3 rounded mr-2" style={{ backgroundColor: '#FF6B35' }}></div>
                      Argo CD
                    </div>
                    <span className="text-gray-400">→</span>
                  </div>
                  <div className="flex items-center justify-between w-full text-left px-3 py-2 text-sm text-gray-700 rounded">
                    <div className="flex items-center">
                      <div className="w-3 h-3 rounded mr-2" style={{ backgroundColor: '#24292E' }}></div>
                      GitHub
                    </div>
                    <span className="text-gray-400">→</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  ),
};