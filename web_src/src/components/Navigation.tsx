import React from 'react';
import { Avatar } from './Avatar/avatar';
import { Dropdown, DropdownButton, DropdownDivider, DropdownHeader, DropdownItem, DropdownLabel, DropdownMenu, DropdownSection } from './Dropdown/dropdown';
import { Text } from './Text/text';
import { MaterialSymbol } from './MaterialSymbol/material-symbol';
import { Link } from './Link/link';
import { useOrganization } from '../hooks/useOrganizationData';
import { useAccount } from '../contexts/AccountContext';
import { useParams } from 'react-router-dom';

const Navigation: React.FC = () => {
  const { account: user } = useAccount();
  const { organizationId } = useParams<{ organizationId: string }>();
  const { data: organization } = useOrganization(organizationId || '');
  return (
    <div className="fixed top-0 left-0 right-0 z-50 bg-white dark:bg-zinc-900 border-zinc-200 dark:border-zinc-800 border-b">
      <div className="flex items-center justify-between px-2 py-[8px]">
        <Link href="/" className="flex items-center flex-shrink-0 text-decoration-none">
          <strong className="ml-2 text-xl text-gray-900 dark:text-white">SuperPlane</strong>
        </Link>
        <div className="flex items-center flex-shrink-0">


          {/* Merged Account Dropdown */}
          <Dropdown>
            <DropdownButton
              plain
              className="flex items-center justify-between gap-x-4 rounded-md border bg-white dark:bg-zinc-950 border-zinc-200 dark:border-zinc-800 text-zinc-700 hover:text-zinc-900 dark:text-zinc-300 dark:hover:text-white"
            >
              {/* Organization Avatar with User Avatar Overlay */}
              <Text className="text-sm font-medium flex-1 text-left">{organization?.metadata?.displayName || organization?.metadata?.name || user?.name}</Text>

              {/* User Avatar (smaller, overlapping in bottom-right) */}
              <Avatar
                src={user?.avatar_url}
                initials={user?.name ? user.name.split(' ').map(n => n[0]).join('').toUpperCase() : '?'}
                alt={user?.name || 'User'}
                className="w-7 h-7"
              />
            </DropdownButton>

            <DropdownMenu className="w-64">
              {/* User Section */}
              <DropdownHeader>
                <div className="flex items-center space-x-3">
                  <Avatar
                    src={user?.avatar_url}
                    initials={user?.name ? user.name.split(' ').map(n => n[0]).join('').toUpperCase() : '?'}
                    alt={user?.name || 'User'}
                    className="size-8"
                  />
                  <div className="flex-1 min-w-0">
                    <Text className="font-medium truncate">{user?.name || 'Loading...'}</Text>
                    <Text className="text-sm text-zinc-500 truncate">{user?.email || 'Loading...'}</Text>
                  </div>
                </div>
              </DropdownHeader>

              {/* User Actions */}
              <DropdownSection>
                <DropdownItem onClick={() => {}}>
                  <span className="flex items-center gap-x-2">
                    <MaterialSymbol name="person" data-slot="icon" size='sm' />
                    <DropdownLabel>Your Profile</DropdownLabel>
                  </span>
                </DropdownItem>

                <DropdownItem onClick={() => {}}>
                  <span className="flex items-center gap-x-2">
                    <MaterialSymbol name="settings" data-slot="icon" size='sm' />
                    <DropdownLabel>Account Settings</DropdownLabel>
                  </span>
                </DropdownItem>
              </DropdownSection>

              {/* Organization Section */}
              <>
                  <DropdownDivider />

                  <DropdownHeader>
                    <div className="flex items-center space-x-3">
                      <Avatar
                        initials={(organization?.metadata?.displayName || organization?.metadata?.name || 'Organization').charAt(0).toUpperCase()}
                        alt={organization?.metadata?.displayName || organization?.metadata?.name || 'Organization'}
                        className="size-8"
                      />
                      <div className="flex-1 min-w-0">
                        <Text className="font-medium truncate">{organization?.metadata?.displayName || organization?.metadata?.name || 'Organization'}</Text>
                      </div>
                    </div>
                  </DropdownHeader>

                  {/* Organization Actions */}
                  <DropdownSection>
                    <DropdownItem href={`/${organizationId}/settings/general`}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="business" data-slot="icon" size='sm' />
                        <DropdownLabel>Organization Settings</DropdownLabel>
                      </span>
                    </DropdownItem>

                    <DropdownItem href={`/${organizationId}/settings/members`}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="person" data-slot="icon" size='sm' />
                        <DropdownLabel>Members</DropdownLabel>
                      </span>
                    </DropdownItem>

                    <DropdownItem href={`/${organizationId}/settings/groups`}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="group" data-slot="icon" size='sm' />
                        <DropdownLabel>Groups</DropdownLabel>
                      </span>
                    </DropdownItem>
                    <DropdownItem href={`/${organizationId}/settings/roles`}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="shield" data-slot="icon" size='sm' />
                        <DropdownLabel>Roles</DropdownLabel>
                      </span>
                    </DropdownItem>

                    <DropdownItem href={`/${organizationId}/settings/billing`}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="credit_card" data-slot="icon" size='sm' />
                        <DropdownLabel>Billing & Plans</DropdownLabel>
                      </span>
                    </DropdownItem>
                  </DropdownSection>
                </>

              <DropdownDivider />

              {/* Sign Out Section */}
              <DropdownSection>
                <DropdownItem onClick={() => {
                  // Redirect to logout
                  window.location.href = '/logout';
                }}>
                  <span className="flex items-center gap-x-2">
                    <MaterialSymbol name="logout" data-slot="icon" size='sm' />
                    <DropdownLabel>Sign Out</DropdownLabel>
                  </span>
                </DropdownItem>
              </DropdownSection>
            </DropdownMenu>
          </Dropdown>
        </div>
      </div>
    </div>
  );
};

export default Navigation;