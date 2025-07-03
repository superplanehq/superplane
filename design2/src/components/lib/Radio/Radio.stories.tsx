import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { Radio, RadioGroup, RadioField } from './radio'
import { Label, Description } from '@headlessui/react'

const meta: Meta<typeof Radio> = {
  title: 'Components/Radio',
  component: Radio,
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
    disabled: { control: 'boolean' },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    value: 'default',
  },
}

export const Colors: Story = {
  render: () => (
    <div className="grid grid-cols-4 gap-4">
      <RadioGroup value="red">
        <Radio value="red" color="red" />
      </RadioGroup>
      <RadioGroup value="orange">
        <Radio value="orange" color="orange" />
      </RadioGroup>
      <RadioGroup value="amber">
        <Radio value="amber" color="amber" />
      </RadioGroup>
      <RadioGroup value="yellow">
        <Radio value="yellow" color="yellow" />
      </RadioGroup>
      <RadioGroup value="lime">
        <Radio value="lime" color="lime" />
      </RadioGroup>
      <RadioGroup value="green">
        <Radio value="green" color="green" />
      </RadioGroup>
      <RadioGroup value="emerald">
        <Radio value="emerald" color="emerald" />
      </RadioGroup>
      <RadioGroup value="teal">
        <Radio value="teal" color="teal" />
      </RadioGroup>
      <RadioGroup value="cyan">
        <Radio value="cyan" color="cyan" />
      </RadioGroup>
      <RadioGroup value="sky">
        <Radio value="sky" color="sky" />
      </RadioGroup>
      <RadioGroup value="blue">
        <Radio value="blue" color="blue" />
      </RadioGroup>
      <RadioGroup value="indigo">
        <Radio value="indigo" color="indigo" />
      </RadioGroup>
      <RadioGroup value="violet">
        <Radio value="violet" color="violet" />
      </RadioGroup>
      <RadioGroup value="purple">
        <Radio value="purple" color="purple" />
      </RadioGroup>
      <RadioGroup value="fuchsia">
        <Radio value="fuchsia" color="fuchsia" />
      </RadioGroup>
      <RadioGroup value="pink">
        <Radio value="pink" color="pink" />
      </RadioGroup>
      <RadioGroup value="rose">
        <Radio value="rose" color="rose" />
      </RadioGroup>
      <RadioGroup value="zinc">
        <Radio value="zinc" color="zinc" />
      </RadioGroup>
    </div>
  ),
}

export const States: Story = {
  render: () => (
    <div className="space-y-4">
      <div className="flex gap-4">
        <RadioGroup value="">
          <Radio value="unchecked" />
        </RadioGroup>
        <RadioGroup value="checked">
          <Radio value="checked" />
        </RadioGroup>
        <RadioGroup value="">
          <Radio value="disabled" disabled />
        </RadioGroup>
        <RadioGroup value="disabled-checked">
          <Radio value="disabled-checked" disabled />
        </RadioGroup>
      </div>
    </div>
  ),
}

export const WithLabels: Story = {
  render: () => {
    const [selected, setSelected] = useState('option1')

    return (
      <RadioGroup value={selected} onChange={setSelected}>
        <RadioField>
          <Radio value="option1" />
          <Label>Option 1</Label>
        </RadioField>
        <RadioField>
          <Radio value="option2" color="blue" />
          <Label>Option 2</Label>
        </RadioField>
        <RadioField>
          <Radio value="option3" color="green" />
          <Label>Option 3</Label>
        </RadioField>
      </RadioGroup>
    )
  },
}

export const WithDescriptions: Story = {
  render: () => {
    const [plan, setPlan] = useState('startup')

    return (
      <RadioGroup value={plan} onChange={setPlan}>
        <RadioField>
          <Radio value="startup" />
          <Label>Startup</Label>
          <Description>12GB / 6 CPUs • 160 GB SSD disk</Description>
        </RadioField>
        <RadioField>
          <Radio value="business" color="blue" />
          <Label>Business</Label>
          <Description>16GB / 8 CPUs • 512 GB SSD disk</Description>
        </RadioField>
        <RadioField>
          <Radio value="enterprise" color="purple" />
          <Label>Enterprise</Label>
          <Description>32GB / 12 CPUs • 1024 GB SSD disk</Description>
        </RadioField>
      </RadioGroup>
    )
  },
}

export const PricingOptions: Story = {
  render: () => {
    const [plan, setPlan] = useState('pro')

    return (
      <div className="w-full max-w-md">
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-4">
          Choose a plan
        </h3>
        <RadioGroup value={plan} onChange={setPlan}>
          <RadioField>
            <Radio value="free" />
            <Label>Free</Label>
            <Description>
              Perfect for trying out our service
              <br />
              <span className="font-semibold">$0/month</span>
            </Description>
          </RadioField>
          <RadioField>
            <Radio value="pro" color="blue" />
            <Label>Pro</Label>
            <Description>
              Best for small teams
              <br />
              <span className="font-semibold">$15/month</span>
            </Description>
          </RadioField>
          <RadioField>
            <Radio value="enterprise" color="purple" />
            <Label>Enterprise</Label>
            <Description>
              For large organizations
              <br />
              <span className="font-semibold">$99/month</span>
            </Description>
          </RadioField>
        </RadioGroup>
      </div>
    )
  },
}

export const PaymentMethods: Story = {
  render: () => {
    const [method, setMethod] = useState('card')

    return (
      <div className="w-full max-w-md">
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-4">
          Payment method
        </h3>
        <RadioGroup value={method} onChange={setMethod}>
          <RadioField>
            <Radio value="card" />
            <Label>Credit Card</Label>
            <Description>Pay with Visa, Mastercard, or American Express</Description>
          </RadioField>
          <RadioField>
            <Radio value="paypal" color="blue" />
            <Label>PayPal</Label>
            <Description>Pay with your PayPal account</Description>
          </RadioField>
          <RadioField>
            <Radio value="bank" color="green" />
            <Label>Bank Transfer</Label>
            <Description>Direct bank transfer (2-3 business days)</Description>
          </RadioField>
          <RadioField>
            <Radio value="crypto" color="orange" />
            <Label>Cryptocurrency</Label>
            <Description>Pay with Bitcoin, Ethereum, or other cryptocurrencies</Description>
          </RadioField>
        </RadioGroup>
      </div>
    )
  },
}

export const SurveyQuestion: Story = {
  render: () => {
    const [satisfaction, setSatisfaction] = useState('')

    return (
      <div className="w-full max-w-lg">
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-2">
          How satisfied are you with our service?
        </h3>
        <p className="text-zinc-600 dark:text-zinc-400 mb-4">
          Please select one option that best describes your experience.
        </p>
        <RadioGroup value={satisfaction} onChange={setSatisfaction}>
          <RadioField>
            <Radio value="very-satisfied" color="green" />
            <Label>Very satisfied</Label>
          </RadioField>
          <RadioField>
            <Radio value="satisfied" color="blue" />
            <Label>Satisfied</Label>
          </RadioField>
          <RadioField>
            <Radio value="neutral" color="zinc" />
            <Label>Neutral</Label>
          </RadioField>
          <RadioField>
            <Radio value="dissatisfied" color="orange" />
            <Label>Dissatisfied</Label>
          </RadioField>
          <RadioField>
            <Radio value="very-dissatisfied" color="red" />
            <Label>Very dissatisfied</Label>
          </RadioField>
        </RadioGroup>
      </div>
    )
  },
}

export const Interactive: Story = {
  render: () => {
    const [theme, setTheme] = useState('system')

    return (
      <div className="space-y-6">
        <div>
          <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-4">
            Theme Preference
          </h3>
          <RadioGroup value={theme} onChange={setTheme}>
            <RadioField>
              <Radio value="light" color="amber" />
              <Label>Light</Label>
              <Description>Light mode theme</Description>
            </RadioField>
            <RadioField>
              <Radio value="dark" />
              <Label>Dark</Label>
              <Description>Dark mode theme</Description>
            </RadioField>
            <RadioField>
              <Radio value="system" color="blue" />
              <Label>System</Label>
              <Description>Follow system preference</Description>
            </RadioField>
          </RadioGroup>
        </div>
        
        <div className="p-4 bg-zinc-100 dark:bg-zinc-800 rounded-lg">
          <p className="text-sm text-zinc-600 dark:text-zinc-400">
            Selected: <strong>{theme}</strong>
          </p>
        </div>
      </div>
    )
  },
}