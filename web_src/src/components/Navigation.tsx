import React from 'react';
import { Avatar } from './Avatar/avatar';
import { Dropdown, DropdownButton, DropdownDivider, DropdownHeader, DropdownItem, DropdownLabel, DropdownMenu, DropdownSection } from './Dropdown/dropdown';
import { Text } from './Text/text';
import { MaterialSymbol } from './MaterialSymbol/material-symbol';
import { Link } from './Link/link';

const Navigation: React.FC = () => {
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
              <Text className="text-sm font-medium flex-1 text-left">TestOrg</Text>

              {/* User Avatar (smaller, overlapping in bottom-right) */}
              <Avatar
                src="https://api.dicebear.com/9.x/initials/svg?seed=Test"
                initials="Test"
                alt="Test"
                className="w-7 h-7 bg-blue-700 dark:bg-blue-900 text-blue-100 dark:text-blue-100"
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

              <DropdownDivider />

              {/* Organization Section */}
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
                  </div>
                </div>
              </DropdownHeader>

              {/* Organization Actions */}
              <DropdownSection>
                <DropdownItem href='/settings'>
                  <span className="flex items-center gap-x-2">
                    <MaterialSymbol name="business" data-slot="icon" size='sm' />
                    <DropdownLabel>Organization Settings</DropdownLabel>
                  </span>
                </DropdownItem>

                <DropdownItem href='/settings/members'>
                  <span className="flex items-center gap-x-2">
                    <MaterialSymbol name="person" data-slot="icon" size='sm' />
                    <DropdownLabel>Members</DropdownLabel>
                  </span>
                </DropdownItem>
                <DropdownItem href='/settings/groups'>
                  <span className="flex items-center gap-x-2">
                    <MaterialSymbol name="group" data-slot="icon" size='sm' />
                    <DropdownLabel>Groups</DropdownLabel>
                  </span>
                </DropdownItem>
                <DropdownItem href='/settings/roles'>
                  <span className="flex items-center gap-x-2">
                    <MaterialSymbol name="shield" data-slot="icon" size='sm' />
                    <DropdownLabel>Roles</DropdownLabel>
                  </span>
                </DropdownItem>

                <DropdownItem href='/settings/billing'>
                  <span className="flex items-center gap-x-2">
                    <MaterialSymbol name="credit_card" data-slot="icon" size='sm' />
                    <DropdownLabel>Billing & Plans</DropdownLabel>
                  </span>
                </DropdownItem>
              </DropdownSection>

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