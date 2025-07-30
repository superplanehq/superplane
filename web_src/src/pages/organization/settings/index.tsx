import { Routes, Route, Navigate, useParams, useNavigate, useLocation } from 'react-router-dom'
import { useEffect } from 'react'
import { Avatar } from '../../../components/Avatar/avatar'
import { Sidebar, SidebarBody, SidebarDivider, SidebarItem, SidebarLabel, SidebarSection } from '../../../components/Sidebar/sidebar'
import { GeneralSettings } from './GeneralSettings'
import { MembersSettings } from './MembersSettings'
import { GroupsSettings } from './GroupsSettings'
import { RolesSettings } from './RolesSettings'
import { GroupMembersPage } from './GroupMembersPage'
import { CreateGroupPage } from './CreateGroupPage'
import { CreateRolePage } from './CreateRolePage'
import { ProfileSettings } from './ProfileSettings'
import { useOrganization } from '../../../hooks/useOrganizationData'
import { useUserStore } from '../../../stores/userStore'

export function OrganizationSettings() {
  const { orgId } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { user, fetchUser } = useUserStore()

  // Use React Query hook for organization data
  const { data: organization, isLoading: loading, error } = useOrganization(orgId || '')

  // Fetch user data when component mounts
  useEffect(() => {
    fetchUser()
  }, [fetchUser])

  // Extract current section from the URL
  const currentSection = location.pathname.split('/').pop() || 'general'

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

  if (error || (!loading && !organization)) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-zinc-500 dark:text-zinc-400">{error instanceof Error ? error.message : 'Organization not found'}</p>
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
                  src={user?.avatar_url}
                  initials={user?.name ? user.name.split(' ').map(n => n[0]).join('').toUpperCase() : 'U'}
                  alt={user?.name || 'My Account'}
                />
                <SidebarLabel className='text-zinc-900 dark:text-white'>{user?.name || 'My Account'}</SidebarLabel>
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
            <SidebarDivider className='dark:border-zinc-800' />
            <SidebarSection>
              <div className='flex items-center gap-3 text-sm font-bold py-3'>
                <Avatar
                  className='w-6 h-6 bg-blue-200 dark:bg-blue-800 text-blue-800 dark:text-white'
                  slot="icon"
                  initials={(organization?.metadata?.displayName || organization?.metadata?.name || orgId).charAt(0).toUpperCase()}
                  alt={organization?.metadata?.displayName || organization?.metadata?.name || orgId}
                />
                <SidebarLabel className='text-zinc-900 dark:text-white'>{organization?.metadata?.displayName || organization?.metadata?.name || orgId}</SidebarLabel>
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
              <Route path="general" element={
                organization ? (
                  <GeneralSettings organization={organization} />
                ) : (
                  <div className="flex justify-center items-center h-32">
                    <p className="text-zinc-500 dark:text-zinc-400">Loading...</p>
                  </div>
                )
              } />
              <Route path="members" element={<MembersSettings organizationId={orgId} />} />
              <Route path="groups" element={<GroupsSettings organizationId={orgId} />} />
              <Route path="roles" element={<RolesSettings organizationId={orgId} />} />
              <Route path="groups/:groupName/members" element={<GroupMembersPage />} />
              <Route path="create-group" element={<CreateGroupPage />} />
              <Route path="create-role" element={<CreateRolePage />} />
              <Route path="create-role/:roleName" element={<CreateRolePage />} />
              <Route path="profile" element={<ProfileSettings />} />
              <Route path="api_token" element={<div className="pt-6"><h1 className="text-2xl font-semibold">API Token</h1><p>API token management coming soon...</p></div>} />
              <Route path="billing" element={<div className="pt-6"><h1 className="text-2xl font-semibold">Billing & Plans</h1><p>Billing management coming soon...</p></div>} />
            </Routes>
          </div>
        </div>
      </div>
    </div>
  )
}