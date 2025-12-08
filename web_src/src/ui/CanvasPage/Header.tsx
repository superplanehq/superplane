import SuperplaneLogo from "@/assets/superplane.svg";
import { Avatar } from "@/components/Avatar/avatar";
import {
  Dropdown,
  DropdownButton,
  DropdownDivider,
  DropdownHeader,
  DropdownItem,
  DropdownLabel,
  DropdownMenu,
  DropdownSection,
} from "@/components/Dropdown/dropdown";
import { Icon } from "@/components/Icon";
import { Text } from "@/components/Text/text";
import { useAccount } from "@/contexts/AccountContext";
import { useOrganization } from "@/hooks/useOrganizationData";
import { resolveIcon } from "@/lib/utils";
import { Save, Undo2 } from "lucide-react";
import { Button } from "../button";

export interface BreadcrumbItem {
  label: string;
  onClick?: () => void;
  href?: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
}

interface HeaderProps {
  breadcrumbs: BreadcrumbItem[];
  onSave?: () => void;
  onUndo?: () => void;
  canUndo?: boolean;
  onLogoClick?: () => void;
  organizationId?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
}

export function Header({
  breadcrumbs,
  onSave,
  onUndo,
  canUndo,
  onLogoClick,
  organizationId,
  unsavedMessage,
  saveIsPrimary,
  saveButtonHidden,
}: HeaderProps) {
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId || "");

  return (
    <>
      <header className="bg-white border-b border-border">
        <div className="relative flex items-center justify-between h-12 px-6">
          {/* Logo */}
          <div className="flex items-center">
            {onLogoClick ? (
              <button
                onClick={onLogoClick}
                className="cursor-pointer hover:opacity-80 transition-opacity"
                aria-label="Go to organization homepage"
              >
                <img src={SuperplaneLogo} alt="Logo" className="w-8 h-8" />
              </button>
            ) : (
              <img src={SuperplaneLogo} alt="Logo" className="w-8 h-8" />
            )}
          </div>

          {/* Breadcrumbs - Absolutely centered */}
          <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 flex items-center space-x-1 text-sm text-gray-500">
            {breadcrumbs.map((item, index) => {
              const IconComponent = item.iconSlug ? resolveIcon(item.iconSlug) : null;

              return (
                <div key={index} className="flex items-center">
                  {index > 0 && <div className="w-2 mx-1">/</div>}
                  {item.href || item.onClick ? (
                    <a
                      href={item.href}
                      onClick={item.onClick}
                      className="hover:text-black transition-colors flex items-center gap-2"
                    >
                      {item.iconSrc && (
                        <div
                          className={`w-5 h-5 rounded-full flex items-center justify-center ${
                            item.iconBackground || ""
                          }`}
                        >
                          <img src={item.iconSrc} alt="" className="w-5 h-5" />
                        </div>
                      )}
                      {IconComponent && (
                        <div
                          className={`w-5 h-5 rounded-full flex items-center justify-center ${
                            item.iconBackground || ""
                          }`}
                        >
                          <IconComponent size={16} className={item.iconColor || ""} />
                        </div>
                      )}
                      {item.label}
                    </a>
                  ) : (
                    <span
                      className={`flex items-center gap-2 ${
                        index === breadcrumbs.length - 1 ? "text-black font-medium" : ""
                      }`}
                    >
                      {item.iconSrc && (
                        <div
                          className={`w-5 h-5 rounded-full flex items-center justify-center ${
                            item.iconBackground || ""
                          }`}
                        >
                          <img src={item.iconSrc} alt="" className="w-5 h-5" />
                        </div>
                      )}
                      {IconComponent && (
                        <div
                          className={`w-5 h-5 rounded-full flex items-center justify-center ${
                            item.iconBackground || ""
                          }`}
                        >
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
            {onUndo && canUndo && (
              <Button onClick={onUndo} size="sm" variant="outline">
                <Undo2 />
                Revert
              </Button>
            )}
            {onSave && !saveButtonHidden && (
              <Button
                onClick={onSave}
                size="sm"
                variant={saveIsPrimary ? "default" : "outline"}
                data-testid="save-canvas-button"
              >
                <Save />
                Save
              </Button>
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
                      initials={
                        account?.name
                          ? account.name
                              .split(" ")
                              .map((n) => n[0])
                              .join("")
                              .toUpperCase()
                          : "?"
                      }
                      alt={account?.name || "User"}
                      className="w-5 h-5"
                    />
                  </DropdownButton>

                  <DropdownMenu className="w-64">
                    {/* User Section */}
                    <DropdownHeader>
                      <div className="flex items-center space-x-3">
                        <Avatar
                          src={account?.avatar_url}
                          initials={
                            account?.name
                              ? account.name
                                  .split(" ")
                                  .map((n) => n[0])
                                  .join("")
                                  .toUpperCase()
                              : "?"
                          }
                          alt={account?.name || "User"}
                          className="size-8"
                        />
                        <div className="flex-1 min-w-0">
                          <Text className="font-medium truncate">{account?.name || "Loading..."}</Text>
                          <Text className="text-sm text-zinc-500 truncate">{account?.email || "Loading..."}</Text>
                        </div>
                      </div>
                    </DropdownHeader>

                    {/* User Actions */}
                    <DropdownSection>
                      <DropdownItem href={`/${organizationId}/settings/profile`}>
                        <span className="flex items-center gap-x-2">
                          <Icon name="person" data-slot="icon" size="sm" />
                          <DropdownLabel>Profile</DropdownLabel>
                        </span>
                      </DropdownItem>
                    </DropdownSection>

                    {/* Organization Section */}
                    <DropdownDivider />

                    <DropdownHeader>
                      <div className="flex items-center space-x-3">
                        <Avatar
                          initials={(organization?.metadata?.name || "Organization").charAt(0).toUpperCase()}
                          alt={organization?.metadata?.name || "Organization"}
                          className="size-8"
                        />
                        <div className="flex-1 min-w-0">
                          <Text className="font-medium truncate max-w-[150px] truncate-ellipsis">
                            {organization?.metadata?.name || "Organization"}
                          </Text>
                        </div>
                      </div>
                    </DropdownHeader>

                    {/* Organization Actions */}
                    <DropdownSection>
                      <DropdownItem href={`/${organizationId}/settings/general`}>
                        <span className="flex items-center gap-x-2">
                          <Icon name="business" data-slot="icon" size="sm" />
                          <DropdownLabel>Organization Settings</DropdownLabel>
                        </span>
                      </DropdownItem>

                      <DropdownItem href={`/${organizationId}/settings/members`}>
                        <span className="flex items-center gap-x-2">
                          <Icon name="person" data-slot="icon" size="sm" />
                          <DropdownLabel>Members</DropdownLabel>
                        </span>
                      </DropdownItem>

                      <DropdownItem href={`/${organizationId}/settings/groups`}>
                        <span className="flex items-center gap-x-2">
                          <Icon name="group" data-slot="icon" size="sm" />
                          <DropdownLabel>Groups</DropdownLabel>
                        </span>
                      </DropdownItem>

                      <DropdownItem href={`/${organizationId}/settings/roles`}>
                        <span className="flex items-center gap-x-2">
                          <Icon name="shield" data-slot="icon" size="sm" />
                          <DropdownLabel>Roles</DropdownLabel>
                        </span>
                      </DropdownItem>

                      <DropdownItem href={`/${organizationId}/settings/integrations`}>
                        <span className="flex items-center gap-x-2">
                          <Icon name="integration_instructions" data-slot="icon" size="sm" />
                          <DropdownLabel>Integrations</DropdownLabel>
                        </span>
                      </DropdownItem>

                      <DropdownItem href="/">
                        <span className="flex items-center gap-x-2">
                          <Icon name="swap_horiz" data-slot="icon" size="sm" />
                          <DropdownLabel>Change organization</DropdownLabel>
                        </span>
                      </DropdownItem>
                    </DropdownSection>

                    <DropdownDivider />

                    {/* Sign Out Section */}
                    <DropdownSection>
                      <DropdownItem
                        onClick={() => {
                          window.location.href = "/logout";
                        }}
                      >
                        <span className="flex items-center gap-x-2">
                          <Icon name="logout" data-slot="icon" size="sm" />
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
    </>
  );
}
