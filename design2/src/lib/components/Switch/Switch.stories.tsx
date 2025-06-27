import type { Meta, StoryObj } from '@storybook/react'
import { useState } from 'react'
import { Switch, SwitchField, SwitchGroup } from './switch'
import { Label, Description } from '@headlessui/react'

const meta: Meta<typeof Switch> = {
  title: 'Components/Switch',
  component: Switch,
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
        'dark',
        'zinc',
        'white',
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
      <Switch checked color="red" />
      <Switch checked color="orange" />
      <Switch checked color="amber" />
      <Switch checked color="yellow" />
      <Switch checked color="lime" />
      <Switch checked color="green" />
      <Switch checked color="emerald" />
      <Switch checked color="teal" />
      <Switch checked color="cyan" />
      <Switch checked color="sky" />
      <Switch checked color="blue" />
      <Switch checked color="indigo" />
      <Switch checked color="violet" />
      <Switch checked color="purple" />
      <Switch checked color="fuchsia" />
      <Switch checked color="pink" />
      <Switch checked color="rose" />
      <Switch checked color="zinc" />
    </div>
  ),
}

export const States: Story = {
  render: () => (
    <div className="space-y-4">
      <div className="flex gap-4 items-center">
        <Switch checked={false} />
        <span className="text-sm text-zinc-600 dark:text-zinc-400">Off</span>
      </div>
      <div className="flex gap-4 items-center">
        <Switch checked={true} />
        <span className="text-sm text-zinc-600 dark:text-zinc-400">On</span>
      </div>
      <div className="flex gap-4 items-center">
        <Switch checked={false} disabled />
        <span className="text-sm text-zinc-600 dark:text-zinc-400">Disabled Off</span>
      </div>
      <div className="flex gap-4 items-center">
        <Switch checked={true} disabled />
        <span className="text-sm text-zinc-600 dark:text-zinc-400">Disabled On</span>
      </div>
    </div>
  ),
}

export const WithLabels: Story = {
  render: () => {
    const [notifications, setNotifications] = useState(true)
    const [darkMode, setDarkMode] = useState(false)
    const [autoSave, setAutoSave] = useState(true)

    return (
      <SwitchGroup>
        <SwitchField>
          <Label>Enable notifications</Label>
          <Switch checked={notifications} onChange={setNotifications} />
        </SwitchField>
        <SwitchField>
          <Label>Dark mode</Label>
          <Switch checked={darkMode} onChange={setDarkMode} color="blue" />
        </SwitchField>
        <SwitchField>
          <Label>Auto-save</Label>
          <Switch checked={autoSave} onChange={setAutoSave} color="green" />
        </SwitchField>
      </SwitchGroup>
    )
  },
}

export const WithDescriptions: Story = {
  render: () => {
    const [emailNotifications, setEmailNotifications] = useState(true)
    const [pushNotifications, setPushNotifications] = useState(false)
    const [analytics, setAnalytics] = useState(true)

    return (
      <SwitchGroup>
        <SwitchField>
          <Label>Email notifications</Label>
          <Switch checked={emailNotifications} onChange={setEmailNotifications} />
          <Description>Receive notifications via email when important events occur.</Description>
        </SwitchField>
        <SwitchField>
          <Label>Push notifications</Label>
          <Switch checked={pushNotifications} onChange={setPushNotifications} color="blue" />
          <Description>Get real-time push notifications on your device.</Description>
        </SwitchField>
        <SwitchField>
          <Label>Analytics tracking</Label>
          <Switch checked={analytics} onChange={setAnalytics} color="purple" />
          <Description>Help us improve by sharing anonymous usage data.</Description>
        </SwitchField>
      </SwitchGroup>
    )
  },
}

export const PrivacySettings: Story = {
  render: () => {
    const [profileVisible, setProfileVisible] = useState(true)
    const [showEmail, setShowEmail] = useState(false)
    const [allowMessages, setAllowMessages] = useState(true)
    const [shareData, setShareData] = useState(false)

    return (
      <div className="w-full max-w-md space-y-6">
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white">
          Privacy Settings
        </h3>
        
        <SwitchGroup>
          <SwitchField>
            <Label>Public profile</Label>
            <Switch checked={profileVisible} onChange={setProfileVisible} color="blue" />
            <Description>Make your profile visible to other users.</Description>
          </SwitchField>
          
          <SwitchField>
            <Label>Show email address</Label>
            <Switch checked={showEmail} onChange={setShowEmail} color="green" />
            <Description>Display your email address on your public profile.</Description>
          </SwitchField>
          
          <SwitchField>
            <Label>Allow direct messages</Label>
            <Switch checked={allowMessages} onChange={setAllowMessages} color="purple" />
            <Description>Let other users send you direct messages.</Description>
          </SwitchField>
          
          <SwitchField>
            <Label>Share usage data</Label>
            <Switch checked={shareData} onChange={setShareData} color="orange" />
            <Description>Help improve our service by sharing anonymous usage statistics.</Description>
          </SwitchField>
        </SwitchGroup>
      </div>
    )
  },
}

export const NotificationPreferences: Story = {
  render: () => {
    const [emailDigest, setEmailDigest] = useState(true)
    const [browserNotifications, setBrowserNotifications] = useState(false)
    const [mobileNotifications, setMobileNotifications] = useState(true)
    const [marketingEmails, setMarketingEmails] = useState(false)

    return (
      <div className="w-full max-w-lg space-y-6">
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white">
          Notification Preferences
        </h3>
        
        <SwitchGroup>
          <SwitchField>
            <Label>Daily email digest</Label>
            <Switch checked={emailDigest} onChange={setEmailDigest} />
            <Description>Receive a daily summary of your activity via email.</Description>
          </SwitchField>
          
          <SwitchField>
            <Label>Browser notifications</Label>
            <Switch checked={browserNotifications} onChange={setBrowserNotifications} color="blue" />
            <Description>Show notifications in your web browser.</Description>
          </SwitchField>
          
          <SwitchField>
            <Label>Mobile push notifications</Label>
            <Switch checked={mobileNotifications} onChange={setMobileNotifications} color="green" />
            <Description>Receive push notifications on your mobile device.</Description>
          </SwitchField>
          
          <SwitchField>
            <Label>Marketing emails</Label>
            <Switch checked={marketingEmails} onChange={setMarketingEmails} color="purple" />
            <Description>Receive promotional emails about new features and offers.</Description>
          </SwitchField>
        </SwitchGroup>
      </div>
    )
  },
}

export const FeatureToggles: Story = {
  render: () => {
    const [betaFeatures, setBetaFeatures] = useState(false)
    const [autoSync, setAutoSync] = useState(true)
    const [offlineMode, setOfflineMode] = useState(false)
    const [advancedMode, setAdvancedMode] = useState(false)

    return (
      <div className="w-full max-w-lg space-y-6">
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white">
          Feature Settings
        </h3>
        
        <SwitchGroup>
          <SwitchField>
            <Label>Beta features</Label>
            <Switch checked={betaFeatures} onChange={setBetaFeatures} color="orange" />
            <Description>Enable experimental features that are still in development.</Description>
          </SwitchField>
          
          <SwitchField>
            <Label>Automatic sync</Label>
            <Switch checked={autoSync} onChange={setAutoSync} color="blue" />
            <Description>Automatically sync your data across all devices.</Description>
          </SwitchField>
          
          <SwitchField>
            <Label>Offline mode</Label>
            <Switch checked={offlineMode} onChange={setOfflineMode} color="green" />
            <Description>Cache data locally for use when you're offline.</Description>
          </SwitchField>
          
          <SwitchField>
            <Label>Advanced mode</Label>
            <Switch checked={advancedMode} onChange={setAdvancedMode} color="purple" />
            <Description>Show advanced options and developer tools.</Description>
          </SwitchField>
        </SwitchGroup>
      </div>
    )
  },
}

export const Interactive: Story = {
  render: () => {
    const [settings, setSettings] = useState({
      notifications: true,
      darkMode: false,
      autoSave: true,
      sharing: false,
    })

    const updateSetting = (key: keyof typeof settings) => {
      setSettings(prev => ({ ...prev, [key]: !prev[key] }))
    }

    return (
      <div className="space-y-6">
        <div>
          <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-4">
            App Settings
          </h3>
          
          <SwitchGroup>
            <SwitchField>
              <Label>Notifications</Label>
              <Switch 
                checked={settings.notifications} 
                onChange={() => updateSetting('notifications')} 
              />
            </SwitchField>
            
            <SwitchField>
              <Label>Dark Mode</Label>
              <Switch 
                checked={settings.darkMode} 
                onChange={() => updateSetting('darkMode')} 
                color="blue" 
              />
            </SwitchField>
            
            <SwitchField>
              <Label>Auto Save</Label>
              <Switch 
                checked={settings.autoSave} 
                onChange={() => updateSetting('autoSave')} 
                color="green" 
              />
            </SwitchField>
            
            <SwitchField>
              <Label>Data Sharing</Label>
              <Switch 
                checked={settings.sharing} 
                onChange={() => updateSetting('sharing')} 
                color="purple" 
              />
            </SwitchField>
          </SwitchGroup>
        </div>
        
        <div className="p-4 bg-zinc-100 dark:bg-zinc-800 rounded-lg">
          <h4 className="font-medium text-zinc-900 dark:text-white mb-2">Current Settings:</h4>
          <pre className="text-sm text-zinc-600 dark:text-zinc-400">
            {JSON.stringify(settings, null, 2)}
          </pre>
        </div>
      </div>
    )
  },
}