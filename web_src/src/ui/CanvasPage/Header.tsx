import SuperplaneLogo from "@/assets/superplane.svg";
import { resolveIcon } from "@/lib/utils";
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "../button";
import { Avatar } from "@/components/Avatar/avatar";
import { Save, Trash2, AlertTriangle } from "lucide-react";
import { Dropdown, DropdownButton, DropdownDivider, DropdownHeader, DropdownItem, DropdownLabel, DropdownMenu, DropdownSection } from "@/components/Dropdown/dropdown";
import { Text } from "@/components/Text/text";
import { MaterialSymbol } from "@/components/MaterialSymbol/material-symbol";
import { useAccount } from "@/contexts/AccountContext";
import { useOrganization } from "@/hooks/useOrganizationData";

export interface BreadcrumbItem {
  label: string;
  onClick?: () => void;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
}

interface HeaderProps {
  breadcrumbs: BreadcrumbItem[];
  onSave?: () => void;
  onDelete?: () => void;
  onLogoClick?: () => void;
  organizationId?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
}

export function Header({ breadcrumbs, onSave, onDelete, onLogoClick, organizationId, unsavedMessage, saveIsPrimary }: HeaderProps) {
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId || '');
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteConfirmation, setDeleteConfirmation] = useState("");

  // Get the workflow name from the last breadcrumb
  const workflowName = breadcrumbs[breadcrumbs.length - 1]?.label || "";

  const handleDeleteClick = () => {
    setShowDeleteModal(true);
    setDeleteConfirmation("");
  };

  const handleConfirmDelete = () => {
    setShowDeleteModal(false);
    setDeleteConfirmation("");
    onDelete?.();
  };

  const isDeleteEnabled = deleteConfirmation === workflowName;

  return (
    <>
      <header className="bg-white border-b border-gray-200">
        <div className="flex items-center justify-between h-12 px-6">
          {/* Logo */}
          <div className="flex items-center">
            {onLogoClick ? (
              <button
                onClick={onLogoClick}
                className="cursor-pointer hover:opacity-80 transition-opacity"
                aria-label="Go to organization homepage"
              >
                <img
                  src={SuperplaneLogo}
                  alt="Logo"
                  className="w-8 h-8"
                />
              </button>
            ) : (
              <img
                src={SuperplaneLogo}
                alt="Logo"
                className="w-8 h-8"
              />
            )}
          </div>

          {/* Breadcrumbs */}
          <div className="flex items-center space-x-2 text-[15px] text-gray-500">
            {breadcrumbs.map((item, index) => {
              const IconComponent = item.iconSlug ? resolveIcon(item.iconSlug) : null;

              return (
                <div key={index} className="flex items-center">
                  {index > 0 && (
                    <div className="w-2 mx-2">/</div>
                  )}
                  {item.onClick ? (
                    <button
                      onClick={item.onClick}
                      className="hover:text-black transition-colors flex items-center gap-2"
                    >
                      {item.iconSrc && (
                        <div className={`w-5 h-5 rounded-full flex items-center justify-center ${item.iconBackground || ""}`}>
                          <img src={item.iconSrc} alt="" className="w-5 h-5" />
                        </div>
                      )}
                      {IconComponent && (
                        <div className={`w-5 h-5 rounded-full flex items-center justify-center ${item.iconBackground || ""}`}>
                          <IconComponent size={16} className={item.iconColor || ""} />
                        </div>
                      )}
                      {item.label}
                    </button>
                  ) : (
                    <span className={`flex items-center gap-2 ${index === breadcrumbs.length - 1 ? "text-black font-medium" : ""}`}>
                      {item.iconSrc && (
                        <div className={`w-5 h-5 rounded-full flex items-center justify-center ${item.iconBackground || ""}`}>
                          <img src={item.iconSrc} alt="" className="w-5 h-5" />
                        </div>
                      )}
                      {IconComponent && (
                        <div className={`w-5 h-5 rounded-full flex items-center justify-center ${item.iconBackground || ""}`}>
                          <IconComponent size={16} className={item.iconColor || ""} />
                        </div>
                      )}
                      {item.label}
                    </span>
                  )}
                </div>
              );
            })}
          </div>

        {/* Right side - Save button and Account Dropdown */}
        <div className="flex items-center gap-3">
          {unsavedMessage && (
            <span className="text-sm text-amber-700 bg-amber-100 px-2 py-1 rounded-md hidden sm:inline">
              {unsavedMessage}
            </span>
          )}
          {onSave && (
            <Button onClick={onSave} size="sm" variant={saveIsPrimary ? "default" : "outline"}>
              <Save />
              Save
            </Button>
          )}
          {onDelete && (
            <button
              onClick={handleDeleteClick}
              className="text-[15px] text-gray-500 hover:text-red-600 transition-colors flex items-center gap-1.5"
            >
              <Trash2 size={16} />
              Delete
            </button>
          )}

          {organizationId && (
            <>
              <div className="h-4 w-px bg-gray-300"></div>
              <Dropdown>
              <DropdownButton
                as="button"
                className="text-[15px] text-gray-500 hover:text-black transition-colors flex items-center gap-2 focus:outline-none"
              >
                <span className="truncate max-w-[150px]">{organization?.metadata?.name || account?.name}</span>
                <Avatar
                  src={account?.avatar_url}
                  initials={account?.name ? account.name.split(' ').map(n => n[0]).join('').toUpperCase() : '?'}
                  alt={account?.name || 'User'}
                  className="w-5 h-5"
                />
              </DropdownButton>

              <DropdownMenu className="w-64">
                {/* User Section */}
                <DropdownHeader>
                  <div className="flex items-center space-x-3">
                    <Avatar
                      src={account?.avatar_url}
                      initials={account?.name ? account.name.split(' ').map(n => n[0]).join('').toUpperCase() : '?'}
                      alt={account?.name || 'User'}
                      className="size-8"
                    />
                    <div className="flex-1 min-w-0">
                      <Text className="font-medium truncate">{account?.name || 'Loading...'}</Text>
                      <Text className="text-sm text-zinc-500 truncate">{account?.email || 'Loading...'}</Text>
                    </div>
                  </div>
                </DropdownHeader>

                {/* User Actions */}
                <DropdownSection>
                  <DropdownItem href={`/${organizationId}/settings/profile`}>
                    <span className="flex items-center gap-x-2">
                      <MaterialSymbol name="person" data-slot="icon" size='sm' />
                      <DropdownLabel>Profile</DropdownLabel>
                    </span>
                  </DropdownItem>
                </DropdownSection>

                {/* Organization Section */}
                <DropdownDivider />

                <DropdownHeader>
                  <div className="flex items-center space-x-3">
                    <Avatar
                      initials={(organization?.metadata?.name || 'Organization').charAt(0).toUpperCase()}
                      alt={organization?.metadata?.name || 'Organization'}
                      className="size-8"
                    />
                    <div className="flex-1 min-w-0">
                      <Text className="font-medium truncate max-w-[150px] truncate-ellipsis">{organization?.metadata?.name || 'Organization'}</Text>
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

                  <DropdownItem href="/">
                    <span className="flex items-center gap-x-2">
                      <MaterialSymbol name="swap_horiz" data-slot="icon" size='sm' />
                      <DropdownLabel>Change organization</DropdownLabel>
                    </span>
                  </DropdownItem>
                </DropdownSection>

                <DropdownDivider />

                {/* Sign Out Section */}
                <DropdownSection>
                  <DropdownItem onClick={() => {
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
            </>
          )}
          </div>
        </div>
      </header>

      {/* Delete Confirmation Modal */}
      <Dialog open={showDeleteModal} onOpenChange={setShowDeleteModal}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <div className="flex items-center gap-3 mb-2">
              <DialogTitle className="text-xl">Delete Workflow</DialogTitle>
            </div>
            <DialogDescription className="text-base space-y-3 pt-2">
              <div className="flex items-center gap-2 px-1 py-2 text-amber-700 bg-amber-50 rounded-lg border border-amber-200">
                <AlertTriangle className="h-4 w-4 flex-shrink-0" />
                <p className="text-sm font-medium">This action cannot be undone.</p>
              </div>
              <p className="text-gray-600">
                This will permanently delete the workflow and all associated events and executions. To proceed, please type the name of the workflow for confirmation.
              </p>
            </DialogDescription>
          </DialogHeader>
          <div className="py-2">
            <Input
              type="text"
              placeholder={workflowName}
              value={deleteConfirmation}
              onChange={(e) => setDeleteConfirmation(e.target.value)}
              autoFocus
            />
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setShowDeleteModal(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={!isDeleteEnabled}
              className="bg-red-600 hover:bg-red-700 text-white disabled:bg-gray-300 disabled:text-gray-500 disabled:cursor-not-allowed"
            >
              <Trash2 className="h-4 w-4 mr-1.5" />
              Delete Workflow
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
