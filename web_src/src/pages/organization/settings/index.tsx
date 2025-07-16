import { Routes, Route, Navigate, useParams, useNavigate, useLocation } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { Avatar } from '../../../components/Avatar/avatar'
import { Sidebar, SidebarBody, SidebarDivider, SidebarItem, SidebarLabel, SidebarSection } from '../../../components/Sidebar/sidebar'
import { GeneralSettings } from './GeneralSettings'
import { MembersSettings } from './MembersSettings'
import { GroupsSettings } from './GroupsSettings'
import { RolesSettings } from './RolesSettings'
import { AddMembersPage } from './AddMembersPage'
import { CreateGroupPage } from './CreateGroupPage'
import { CreateRolePage } from './CreateRolePage'
import { organizationsDescribeOrganization } from '../../../api-client/sdk.gen'
import type { OrganizationsOrganization } from '../../../api-client/types.gen'

export function OrganizationSettings() {
  const { orgId } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [organization, setOrganization] = useState<OrganizationsOrganization | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  
  // Extract current section from the URL
  const currentSection = location.pathname.split('/').pop() || 'general'
  
  // Fetch organization details
  useEffect(() => {
    if (!orgId) return
    
    const fetchOrganization = async () => {
      try {
        setLoading(true)
        const response = await organizationsDescribeOrganization({
          path: { idOrName: orgId }
        })
        setOrganization(response.data?.organization || null)
      } catch (err) {
        setError('Failed to load organization')
        console.error('Error fetching organization:', err)
      } finally {
        setLoading(false)
      }
    }
    
    fetchOrganization()
  }, [orgId])

  if (!orgId) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-zinc-500 dark:text-zinc-400">Organization ID not found</p>
      </div>
    )
  }
  
  if (loading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-zinc-500 dark:text-zinc-400">Loading organization...</p>
      </div>
    )
  }
  
  if (error || !organization) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-zinc-500 dark:text-zinc-400">{error || 'Organization not found'}</p>
      </div>
    )
  }

  const tabs = [
    { id: 'profile', label: 'Profile', icon: 'person' },
    { id: 'general', label: 'General', icon: 'settings' },
    { id: 'members', label: 'Members', icon: 'group' },
    { id: 'groups', label: 'Groups', icon: 'group' },
    { id: 'roles', label: 'Roles', icon: 'admin_panel_settings' }
  ]


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
              <SidebarItem className={`${currentSection === 'profile' ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`} onClick={() => navigate(`/organization/${orgId}/settings/profile`)}>
                <span className='px-7'>
                  <SidebarLabel>My Profile</SidebarLabel>
                </span>
              </SidebarItem>
              <SidebarItem className={`${currentSection === 'api_token' ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`} onClick={() => navigate(`/organization/${orgId}/settings/api_token`)}>
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
                  initials={(organization.metadata?.displayName || organization.metadata?.name || orgId).charAt(0).toUpperCase()}
                  alt={organization.metadata?.displayName || organization.metadata?.name || orgId}
                />
                <SidebarLabel className='text-zinc-900 dark:text-white'>{organization.metadata?.displayName || organization.metadata?.name || orgId}</SidebarLabel>
              </div>
              {tabs.filter(tab => tab.id !== 'profile').map((tab) => (
                <SidebarItem 
                  key={tab.id} 
                  onClick={() => navigate(`/organization/${orgId}/settings/${tab.id}`)} 
                  className={`${currentSection === tab.id ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`}
                >
                  <span className={`px-7 ${currentSection === tab.id ? 'font-semibold' : 'font-normal'}`}>
                    <SidebarLabel>{tab.label}</SidebarLabel>
                  </span>
                </SidebarItem>
              ))}
            </SidebarSection>
          </SidebarBody>
        </Sidebar>
        
        <div className="flex-1 overflow-auto bg-zinc-50 dark:bg-zinc-900">
          <div className="px-8 pb-8">
            <Routes>
              <Route path="" element={<Navigate to="general" replace />} />
              <Route path="general" element={<GeneralSettings organization={organization} />} />
              <Route path="members" element={<MembersSettings organizationId={orgId} />} />
              <Route path="groups" element={<GroupsSettings organizationId={orgId} />} />
              <Route path="roles" element={<RolesSettings organizationId={orgId} />} />
              <Route path="add-members" element={<AddMembersPage />} />
              <Route path="create-group" element={<CreateGroupPage />} />
              <Route path="create-role" element={<CreateRolePage />} />
              <Route path="create-role/:roleName" element={<CreateRolePage />} />
              <Route path="profile" element={<div className="pt-6"><h1 className="text-2xl font-semibold">Profile Settings</h1><p>Profile settings coming soon...</p></div>} />
              <Route path="api_token" element={<div className="pt-6"><h1 className="text-2xl font-semibold">API Token</h1><p>API token management coming soon...</p></div>} />
              <Route path="billing" element={<div className="pt-6"><h1 className="text-2xl font-semibold">Billing & Plans</h1><p>Billing management coming soon...</p></div>} />
            </Routes>
          </div>
        </div>
      </div>
    </div>
  )
}