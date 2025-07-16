import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { Avatar } from './Avatar/avatar';
import { Dropdown, DropdownButton, DropdownDivider, DropdownHeader, DropdownItem, DropdownLabel, DropdownMenu, DropdownSection } from './Dropdown/dropdown';
import { Text } from './Text/text';
import { MaterialSymbol } from './MaterialSymbol/material-symbol';
import { Link } from './Link/link';
import { organizationsDescribeOrganization } from '../api-client/sdk.gen';
import type { OrganizationsOrganization } from '../api-client/types.gen';

const Navigation: React.FC = () => {
  const { orgId } = useParams<{ orgId?: string }>();
  const [organization, setOrganization] = useState<OrganizationsOrganization | null>(null);
  
  useEffect(() => {
    if (!orgId) return;
    
    const fetchOrganization = async () => {
      try {
        const response = await organizationsDescribeOrganization({
          path: { idOrName: orgId }
        });
        setOrganization(response.data?.organization || null);
      } catch (err) {
        console.error('Error fetching organization:', err);
      }
    };
    
    fetchOrganization();
  }, [orgId]);
  return (
    <div className="fixed top-0 left-0 right-0 z-50 bg-white shadow-sm">
      <div className="flex items-center justify-between px-2 py-[8px]">
        <Link href="/" className="flex items-center flex-shrink-0 text-decoration-none">
          <strong className="ml-2 text-xl text-gray-900">SuperPlane</strong>
        </Link>
        <div className="flex items-center flex-shrink-0">


          {/* Merged Account Dropdown */}
          <Dropdown>
            <DropdownButton
              plain
              className="flex items-center justify-between gap-x-4 rounded-md border bg-white dark:bg-zinc-950 border-zinc-200 dark:border-zinc-800 text-zinc-700 hover:text-zinc-900 dark:text-zinc-300 dark:hover:text-white"
            >
              {/* Organization Avatar with User Avatar Overlay */}
              <Text className="text-sm font-medium flex-1 text-left">{organization?.metadata?.displayName || organization?.metadata?.name || 'Organization'}</Text>

              {/* User Avatar (smaller, overlapping in bottom-right) */}
              <Avatar
                initials={(organization?.metadata?.displayName || organization?.metadata?.name || 'Organization').charAt(0).toUpperCase()}
                alt={organization?.metadata?.displayName || organization?.metadata?.name || 'Organization'}
                className="w-7 h-7"
              />
            </DropdownButton>

            <DropdownMenu className="w-64">
              {/* User Section */}
              <DropdownHeader>
                <div className="flex items-center space-x-3">
                  <Avatar
                    src="https://api.dicebear.com/9.x/initials/svg?seed=Test"
                    initials="Test"
                    alt="Test"
                    className="size-8"
                  />
                  <div className="flex-1 min-w-0">
                    <Text className="font-medium truncate">Test</Text>
                    <Text className="text-sm text-zinc-500 truncate">TestUsername</Text>
                  </div>
                </div>
              </DropdownHeader>

              {/* User Actions */}
              <DropdownSection>
                <DropdownItem onClick={() => console.log('Profile')}>
                  <span className="flex items-center gap-x-2">
                    <MaterialSymbol name="person" data-slot="icon" size='sm' />
                    <DropdownLabel>Your Profile</DropdownLabel>
                  </span>
                </DropdownItem>

                <DropdownItem onClick={() => console.log('Settings')}>
                  <span className="flex items-center gap-x-2">
                    <MaterialSymbol name="settings" data-slot="icon" size='sm' />
                    <DropdownLabel>Account Settings</DropdownLabel>
                  </span>
                </DropdownItem>
              </DropdownSection>

              {/* Organization Section - Only show when in organization context */}
              {orgId && (
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
                    <DropdownItem href={`/organization/${orgId}/settings/general`}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="business" data-slot="icon" size='sm' />
                        <DropdownLabel>Organization Settings</DropdownLabel>
                      </span>
                    </DropdownItem>

                    <DropdownItem href={`/organization/${orgId}/settings/members`}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="person" data-slot="icon" size='sm' />
                        <DropdownLabel>Members</DropdownLabel>
                      </span>
                    </DropdownItem>
                    <DropdownItem href={`/organization/${orgId}/settings/groups`}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="group" data-slot="icon" size='sm' />
                        <DropdownLabel>Groups</DropdownLabel>
                      </span>
                    </DropdownItem>
                    <DropdownItem href={`/organization/${orgId}/settings/roles`}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="shield" data-slot="icon" size='sm' />
                        <DropdownLabel>Roles</DropdownLabel>
                      </span>
                    </DropdownItem>

                    <DropdownItem href={`/organization/${orgId}/settings/billing`}>
                      <span className="flex items-center gap-x-2">
                        <MaterialSymbol name="credit_card" data-slot="icon" size='sm' />
                        <DropdownLabel>Billing & Plans</DropdownLabel>
                      </span>
                    </DropdownItem>
                  </DropdownSection>
                </>
              )}

              <DropdownDivider />

              {/* Sign Out Section */}
              <DropdownSection>
                <DropdownItem onClick={() => console.log('Sign out')}>
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