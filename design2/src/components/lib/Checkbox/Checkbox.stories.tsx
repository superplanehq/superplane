import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { Checkbox, CheckboxField, CheckboxGroup } from './checkbox'
import { Label } from '@headlessui/react'
import { Text } from '../Text/text'

const meta: Meta<typeof Checkbox> = {
  title: 'Components/Checkbox',
  component: Checkbox,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    color: {
      control: 'select',
      options: [
        'dark/zinc',
        'dark/white',
        'white',
        'dark',
        'zinc',
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
      ],
    },
    checked: { control: 'boolean' },
    disabled: { control: 'boolean' },
    indeterminate: { control: 'boolean' },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    checked: false,
  },
}

export const Colors: Story = {
  render: () => (
    <div className="grid grid-cols-4 gap-4">
      <Checkbox color="red" defaultChecked />
      <Checkbox color="orange" defaultChecked />
      <Checkbox color="amber" defaultChecked />
      <Checkbox color="yellow" defaultChecked />
      <Checkbox color="lime" defaultChecked />
      <Checkbox color="green" defaultChecked />
      <Checkbox color="emerald" defaultChecked />
      <Checkbox color="teal" defaultChecked />
      <Checkbox color="cyan" defaultChecked />
      <Checkbox color="sky" defaultChecked />
      <Checkbox color="blue" defaultChecked />
      <Checkbox color="indigo" defaultChecked />
      <Checkbox color="violet" defaultChecked />
      <Checkbox color="purple" defaultChecked />
      <Checkbox color="fuchsia" defaultChecked />
      <Checkbox color="pink" defaultChecked />
      <Checkbox color="rose" defaultChecked />
      <Checkbox color="zinc" defaultChecked />
    </div>
  ),
}

export const States: Story = {
  render: () => (
    <div className="space-y-4">
      <div className="flex gap-4">
        <Checkbox />
        <Checkbox defaultChecked />
        <Checkbox indeterminate />
      </div>
      <div className="flex gap-4">
        <Checkbox disabled />
        <Checkbox disabled defaultChecked />
        <Checkbox disabled indeterminate />
      </div>
    </div>
  ),
}

export const WithLabels: Story = {
  render: () => (
    <CheckboxGroup>
      <CheckboxField>
        <Checkbox name="notifications" />
        <Label>Enable notifications</Label>
      </CheckboxField>
      <CheckboxField>
        <Checkbox name="marketing" color="blue" />
        <Label>Marketing emails</Label>
      </CheckboxField>
      <CheckboxField>
        <Checkbox name="updates" color="green" />
        <Label>Product updates</Label>
      </CheckboxField>
    </CheckboxGroup>
  ),
}

export const WithDescriptions: Story = {
  render: () => (
    <CheckboxGroup>
      <CheckboxField>
        <Checkbox name="notifications" />
        <Label>Push notifications</Label>
        <Text slot="description">Get notified when someone mentions you in a comment.</Text>
      </CheckboxField>
      <CheckboxField>
        <Checkbox name="comments" color="blue" />
        <Label>Comments</Label>
        <Text slot="description">Get notified when someone posts a comment on a project.</Text>
      </CheckboxField>
      <CheckboxField>
        <Checkbox name="reminders" color="green" />
        <Label>Reminders</Label>
        <Text slot="description">Get notified about upcoming events and deadlines.</Text>
      </CheckboxField>
    </CheckboxGroup>
  ),
}

export const Interactive: Story = {
  render: () => {
    const [checked, setChecked] = useState(false)
    const [indeterminate, setIndeterminate] = useState(false)

    return (
      <div className="space-y-4">
        <CheckboxField>
          <Checkbox 
            checked={checked} 
            indeterminate={indeterminate}
            onChange={setChecked}
          />
          <Label>Interactive checkbox</Label>
        </CheckboxField>
        <div className="flex gap-2">
          <button 
            className="px-3 py-1 bg-blue-500 text-white rounded text-sm"
            onClick={() => {
              setChecked(!checked)
              setIndeterminate(false)
            }}
          >
            Toggle
          </button>
          <button 
            className="px-3 py-1 bg-gray-500 text-white rounded text-sm"
            onClick={() => {
              setIndeterminate(!indeterminate)
              setChecked(false)
            }}
          >
            Indeterminate
          </button>
        </div>
      </div>
    )
  },
}

export const FormExample: Story = {
  render: () => {
    const [preferences, setPreferences] = useState({
      newsletter: true,
      promotions: false,
      updates: false,
      notifications: true,
    })

    return (
      <div className="space-y-6">
        <h3 className="text-lg font-semibold">Email Preferences</h3>
        <CheckboxGroup>
          <CheckboxField>
            <Checkbox 
              checked={preferences.newsletter}
              onChange={(checked) => setPreferences(prev => ({ ...prev, newsletter: checked }))}
            />
            <Label>Newsletter</Label>
            <Text slot="description">Weekly updates about new features and improvements.</Text>
          </CheckboxField>
          <CheckboxField>
            <Checkbox 
              color="orange"
              checked={preferences.promotions}
              onChange={(checked) => setPreferences(prev => ({ ...prev, promotions: checked }))}
            />
            <Label>Promotions</Label>
            <Text slot="description">Special offers and discounts on our products.</Text>
          </CheckboxField>
          <CheckboxField>
            <Checkbox 
              color="blue"
              checked={preferences.updates}
              onChange={(checked) => setPreferences(prev => ({ ...prev, updates: checked }))}
            />
            <Label>Product Updates</Label>
            <Text slot="description">Notifications about new features and bug fixes.</Text>
          </CheckboxField>
          <CheckboxField>
            <Checkbox 
              color="green"
              checked={preferences.notifications}
              onChange={(checked) => setPreferences(prev => ({ ...prev, notifications: checked }))}
            />
            <Label>Push Notifications</Label>
            <Text slot="description">Real-time notifications in your browser.</Text>
          </CheckboxField>
        </CheckboxGroup>
      </div>
    )
  },
}