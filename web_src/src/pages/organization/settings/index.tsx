import { useState } from 'react'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Avatar } from '../../../components/Avatar/avatar'
import { Heading } from '../../../components/Heading/heading'
import { Button } from '../../../components/Button/button'
import { Input, InputGroup } from '../../../components/Input/input'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem,
  DropdownLabel,
  DropdownDescription
} from '../../../components/Dropdown/dropdown'
import { Link } from '../../../components/Link/link'
import { Field, Fieldset, Label } from '../../../components/Fieldset/fieldset'
import { Textarea } from '../../../components/Textarea/textarea'
import { AddMembersSection } from './AddMembersSection'
import { 
  Table, 
  TableHead, 
  TableBody, 
  TableRow, 
  TableHeader, 
  TableCell 
} from '../../../components/Table/table'
import { Sidebar, SidebarBody, SidebarDivider, SidebarItem, SidebarLabel, SidebarSection } from '../../../components/Sidebar/sidebar'

export function OrganizationSettings() {
  const [activeTab, setActiveTab] = useState<'profile' | 'general' | 'members' | 'groups' | 'roles' | 'tokens' | 'integrations' | 'api_token' | 'security'>('general')
  
  const currentOrganization = {
    id: '1',
    name: 'Confluent',
    avatar: 'https://confluent.io/favicon.ico',
    initials: 'C'
  }

  const tabs = [
    { id: 'profile', label: 'Profile', icon: 'person' },
    { id: 'general', label: 'General', icon: 'settings' },
    { id: 'members', label: 'Members', icon: 'group' },
    { id: 'groups', label: 'Groups', icon: 'group' },
    { id: 'roles', label: 'Roles', icon: 'admin_panel_settings' }
  ]

  const renderTabContent = () => {
    switch (activeTab) {
      case 'general':
        return (
          <div className="space-y-6 pt-6">
            <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
              General
            </Heading>
            <Fieldset className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 space-y-6 max-w-xl">
              <Field>
                <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                  Organization Name
                </Label>
                <Input
                  type="text"
                  defaultValue={currentOrganization.name}
                  className="max-w-lg"
                />
              </Field>
              <Field>
                <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                  Description
                </Label>
                <Textarea
                  placeholder="Enter organization description"
                  className="max-w-lg"
                />
              </Field>
              
              <Field>
                <div className="flex items-start gap-4">
                  <div className='w-1/2 flex-col gap-2'>
                    <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                      Company Logo
                    </Label>
                    <div className="flex-none grow-0">
                      <div className="inline-block h-15 py-4 bg-white dark:bg-zinc-700 rounded-lg border border-zinc-200 dark:border-zinc-600 border-dashed px-4">  
                        <img
                          src="https://upload.wikimedia.org/wikipedia/commons/a/ab/Confluent%2C_Inc._logo.svg"
                          alt="Confluent, Inc."
                          className='h-full'
                        />
                      </div>
                      <div className="flex items-center gap-2">
                        <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                          Upload new 
                        </Link>
                        <span className="text-xs text-zinc-500 dark:text-zinc-400">
                          &bull;
                        </span>
                        <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                          Remove  
                        </Link>
                      </div>
                      <p className="text-xs text-zinc-500 dark:text-zinc-400">
                        Rectangle image 96X20px
                      </p>
                    </div>
                  </div>
                  <div className='w-1/2 flex-col gap-2'>
                    <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                      Company Icon
                    </Label> 
                    <div className="flex-none grow-0">
                      <div className="w-15 h-15 inline-block py-4 bg-white dark:bg-zinc-700 rounded-lg border border-zinc-200 dark:border-zinc-600 border-dashed px-4">
                        <img
                          src="https://confluent.io/favicon.ico"
                          alt="Confluent, Inc."
                          height={24}
                        />
                      </div>
                    </div>
                    <div className="flex flex-col">
                      <div className="flex items-center gap-2">
                        <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                          Upload new 
                        </Link>
                        <span className="text-xs text-zinc-500 dark:text-zinc-400">
                          &bull;
                        </span>
                        <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                          Remove  
                        </Link>
                      </div>
                      <p className="text-xs text-zinc-500 dark:text-zinc-400">
                        Square image 64X64px
                      </p>
                    </div>
                  </div>
                </div>
              </Field>
            </Fieldset>
          </div>
        )
        
      case 'members':
        return (
          <div className="space-y-6 pt-6">
            <div className="flex items-center justify-between">
              <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                Members
              </Heading>
            </div>
            <AddMembersSection />
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
              <div className="px-6 pt-6 pb-4">
                <div className="flex items-center justify-between">
                  <InputGroup>
                    <Input name="search" placeholder="Search members…" aria-label="Search" className="w-xs" />
                  </InputGroup>
                </div>
              </div>
              <div className="px-6 pb-6">
                <Table dense>
                  <TableHead>
                    <TableRow>
                      <TableHeader>Name</TableHeader>
                      <TableHeader>Email</TableHeader>
                      <TableHeader>Role</TableHeader>
                      <TableHeader>Status</TableHeader>
                      <TableHeader></TableHeader>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    <TableRow>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <Avatar initials="JD" className="size-8" />
                          <div>
                            <div className="text-sm font-medium text-zinc-900 dark:text-white">
                              John Doe
                            </div>
                            <div className="text-xs text-zinc-500 dark:text-zinc-400">
                              Last active: 2 hours ago
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>john@acme.com</TableCell>
                      <TableCell>
                        <Dropdown>
                          <DropdownButton outline className="flex items-center gap-2 text-sm">
                            Owner
                            <MaterialSymbol name="keyboard_arrow_down" />
                          </DropdownButton>
                          <DropdownMenu>
                            <DropdownItem>
                              <DropdownLabel>Owner</DropdownLabel>
                              <DropdownDescription>Owner role description.</DropdownDescription>
                            </DropdownItem>
                            <DropdownItem>
                              <DropdownLabel>Admin</DropdownLabel>
                              <DropdownDescription>Admin role description.</DropdownDescription>
                            </DropdownItem>
                            <DropdownItem>
                              <DropdownLabel>Member</DropdownLabel>
                              <DropdownDescription>Member role description.</DropdownDescription>
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                      </TableCell>
                      <TableCell>
                        <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400">
                          Active
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end">
                          <Dropdown>
                            <DropdownButton plain className="flex items-center gap-2 text-sm">
                              <MaterialSymbol name="more_vert" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem>
                                <MaterialSymbol name="edit" />
                                Edit
                              </DropdownItem>
                              <DropdownItem>
                                <MaterialSymbol name="block" />
                                Suspend
                              </DropdownItem>
                              <DropdownItem>
                                <MaterialSymbol name="delete" />
                                Remove
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </div>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        )
        
      case 'groups':
        return (
          <div className="space-y-6 pt-6">
            <div className="flex items-center justify-between">
              <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                Groups
              </Heading>
            </div>
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
              <div className="px-6 pt-6 pb-4 flex items-center justify-between">
                <InputGroup>
                  <Input name="search" placeholder="Search Groups…" aria-label="Search" className="w-xs" />
                </InputGroup>
                <Button color="blue" className='flex items-center'>
                  <MaterialSymbol name="add" />
                  Create New Group
                </Button>
              </div>
              <div className="px-6 pb-6">
                <Table dense>
                  <TableHead>
                    <TableRow>
                      <TableHeader>Team name</TableHeader>
                      <TableHeader>Created</TableHeader>
                      <TableHeader>Members</TableHeader>
                      <TableHeader>Role</TableHeader>
                      <TableHeader></TableHeader>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    <TableRow>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <Avatar className='w-9' square initials="E" />
                          <div>
                            <Link href="#" className="cursor-pointer text-sm font-medium text-blue-600 dark:text-blue-400">
                              Engineering
                            </Link>
                            <div className="text-xs text-zinc-500 dark:text-zinc-400">
                              Software development and technical operations
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>2 months ago</TableCell>
                      <TableCell>8 members</TableCell>
                      <TableCell>
                        <Dropdown>
                          <DropdownButton outline className="flex items-center gap-2 text-sm">
                            Admin
                            <MaterialSymbol name="keyboard_arrow_down" />
                          </DropdownButton>
                          <DropdownMenu>
                            <DropdownItem>
                              <DropdownLabel>Admin</DropdownLabel>
                              <DropdownDescription>Admin role description.</DropdownDescription>
                            </DropdownItem>
                            <DropdownItem>
                              <DropdownLabel>Member</DropdownLabel>
                              <DropdownDescription>Member role description.</DropdownDescription>
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end">
                          <Dropdown>
                            <DropdownButton plain>
                              <MaterialSymbol name="more_vert" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem>
                                <MaterialSymbol name="group" />
                                View Members
                              </DropdownItem>
                              <DropdownItem>
                                <MaterialSymbol name="edit" />
                                Edit Team
                              </DropdownItem>
                              <DropdownItem>
                                <MaterialSymbol name="delete" />
                                Delete Team
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </div>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        )
        
      case 'roles':
        return (
          <div className="space-y-6 pt-6">
            <div className="flex items-center justify-between">
              <div>
                <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-1">
                  Roles
                </Heading>
              </div>
            </div>
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
              <div className="px-6 pt-6 pb-4 flex items-center justify-between">
                <InputGroup>
                  <Input name="search" placeholder="Search Roles…" aria-label="Search" className="w-xs" />
                </InputGroup>
                <Button color="blue" className='flex items-center'>
                  <MaterialSymbol name="add" />
                  New role
                </Button>
              </div>
              <div className="px-6 pb-6">
                <Table dense>
                  <TableHead>
                    <TableRow>
                      <TableHeader>Role name</TableHeader>
                      <TableHeader>Permissions</TableHeader>
                      <TableHeader>Status</TableHeader>
                      <TableHeader></TableHeader>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    <TableRow>
                      <TableCell className="font-medium">Admin</TableCell>
                      <TableCell>8</TableCell>
                      <TableCell>
                        <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400">
                          Active
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end">
                          <Dropdown>
                            <DropdownButton plain>
                              <MaterialSymbol name="more_vert" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem>
                                <MaterialSymbol name="edit" />
                                Edit
                              </DropdownItem>
                              <DropdownItem>
                                <MaterialSymbol name="copy" />
                                Duplicate
                              </DropdownItem>
                              <DropdownItem>
                                <MaterialSymbol name="delete" />
                                Delete
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </div>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        )
        
      default:
        return null
    }
  }

  return (
    <div className="flex flex-col bg-zinc-50 dark:bg-zinc-950" style={{ height: "calc(100vh - 3rem)", marginTop: "3rem" }}>
      <div className="flex flex-1 overflow-hidden">
        <Sidebar className='w-70 bg-white dark:bg-zinc-950 border-r bw-1 border-zinc-200 dark:border-zinc-800'>
          <SidebarBody>
            <SidebarSection>
              <div className='flex items-center gap-3 text-sm font-bold py-3'>
                <Avatar 
                  className='w-6 h-6'
                  src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face"
                  alt="My Account"
                />
                <SidebarLabel className='text-zinc-900 dark:text-white'>My Account</SidebarLabel>
              </div>
              <SidebarItem className={`${activeTab === 'profile' ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`} onClick={() => setActiveTab('profile')}>
                <span className='px-7'>
                  <SidebarLabel>My Profile</SidebarLabel>
                </span>
              </SidebarItem>
              <SidebarItem className={`${activeTab === 'api_token' ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`} onClick={() => setActiveTab('api_token')}>
                <span className='px-7'>
                  <SidebarLabel>API Token</SidebarLabel>
                </span>
              </SidebarItem>
            </SidebarSection>
            <SidebarDivider className='dark:border-zinc-800'/>
            <SidebarSection>
              <div className='flex items-center gap-3 text-sm font-bold py-3'>
                <Avatar 
                  className='w-6 h-6'
                  slot="icon"
                  src="https://www.confluent.io/favicon.ico"
                  alt="Confluent"
                />
                <SidebarLabel className='text-zinc-900 dark:text-white'>Confluent</SidebarLabel>
              </div>
              {tabs.filter(tab => tab.id !== 'profile').map((tab) => (
                <SidebarItem 
                  key={tab.id} 
                  onClick={() => setActiveTab(tab.id as any)} 
                  className={`${activeTab === tab.id ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`}
                >
                  <span className={`px-7 ${activeTab === tab.id ? 'font-semibold' : 'font-normal'}`}>
                    <SidebarLabel>{tab.label}</SidebarLabel>
                  </span>
                </SidebarItem>
              ))}
            </SidebarSection>
          </SidebarBody>
        </Sidebar>
        
        <div className="flex-1 overflow-auto bg-zinc-50 dark:bg-zinc-900">
          <div className="px-8 pb-8">
            {renderTabContent()}
          </div>
        </div>
      </div>
    </div>
  )
}