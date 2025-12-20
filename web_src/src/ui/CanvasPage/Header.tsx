import SuperplaneLogo from "@/assets/superplane.svg";
// import { Avatar } from "@/components/Avatar/avatar";
import { Icon } from "@/components/Icon";
import { useAccount } from "@/contexts/AccountContext";
import { useOrganization } from "@/hooks/useOrganizationData";
import { resolveIcon } from "@/lib/utils";
import { Undo2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
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
  const forceSidebarVisible = false;
  const [isSidebarOpen, setIsSidebarOpen] = useState(forceSidebarVisible);
  const sidebarTimeoutRef = useRef<number | null>(null);

  const clearSidebarTimeout = () => {
    if (sidebarTimeoutRef.current !== null) {
      window.clearTimeout(sidebarTimeoutRef.current);
      sidebarTimeoutRef.current = null;
    }
  };

  const openSidebar = () => {
    clearSidebarTimeout();
    setIsSidebarOpen(true);
  };

  const scheduleSidebarClose = () => {
    if (forceSidebarVisible) return;
    clearSidebarTimeout();
    sidebarTimeoutRef.current = window.setTimeout(() => {
      setIsSidebarOpen(false);
    }, 150);
  };

  const handleLogoMouseEnter = () => {
    if (forceSidebarVisible) return;
    if (!organizationId) return;
    openSidebar();
  };

  const handleLogoMouseLeave = () => {
    if (forceSidebarVisible) return;
    if (!organizationId) return;
    scheduleSidebarClose();
  };

  const handleSidebarMouseEnter = () => {
    if (forceSidebarVisible) return;
    if (!organizationId) return;
    openSidebar();
  };

  const handleSidebarMouseLeave = () => {
    if (forceSidebarVisible) return;
    if (!organizationId) return;
    scheduleSidebarClose();
  };

  useEffect(() => {
    return () => clearSidebarTimeout();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // const accountInitials = account?.name
  //   ? account.name
  //       .split(" ")
  //       .map((n) => n[0])
  //       .join("")
  //       .toUpperCase()
  //   : "?";

  const organizationName = organization?.metadata?.name || "Organization";

  const sidebarUserLinks = [
    {
      label: "Profile",
      href: organizationId ? `/${organizationId}/settings/profile` : "#",
      icon: "person",
    },
    {
      label: "Sign Out",
      icon: "logout",
      onClick: () => handleSignOut(),
    },
  ];

  const sidebarOrganizationLinks = [
    {
      label: "Settings",
      href: organizationId ? `/${organizationId}/settings/general` : "#",
      icon: "business",
    },
    { label: "Members", href: organizationId ? `/${organizationId}/settings/members` : "#", icon: "person" },
    { label: "Groups", href: organizationId ? `/${organizationId}/settings/groups` : "#", icon: "group" },
    { label: "Roles", href: organizationId ? `/${organizationId}/settings/roles` : "#", icon: "shield" },
    {
      label: "Integrations",
      href: organizationId ? `/${organizationId}/settings/integrations` : "#",
      icon: "integration_instructions",
    },
    { label: "Change Organization", href: "/", icon: "swap_horiz" },
  ];

  const handleSignOut = () => {
    window.location.href = "/logout";
  };

  return (
    <>
      <header className="bg-white border-b border-border">
        <div className="relative flex items-center justify-between h-12 px-4">
          {/* Logo */}
          <div className="flex items-center" onMouseEnter={handleLogoMouseEnter} onMouseLeave={handleLogoMouseLeave}>
            {onLogoClick ? (
              <button onClick={onLogoClick} className="cursor-pointer" aria-label="Go to organization homepage">
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

          {/* Right side - Save button */}
          <div className="flex items-center gap-3">
            {unsavedMessage && (
              <span className="text-sm text-yellow-700 bg-orange-100 px-2 py-1 rounded-md hidden sm:inline">
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
                Save
              </Button>
            )}
          </div>
        </div>
      </header>
      {organizationId && (
        <div
          className={`fixed inset-y-0 left-0 z-[60] w-60 border-r border-border bg-white shadow-lg transition-all duration-200 ease-in-out ${
            isSidebarOpen ? "translate-x-0 opacity-100" : "-translate-x-full opacity-0 pointer-events-none"
          }`}
          onMouseEnter={handleSidebarMouseEnter}
          onMouseLeave={handleSidebarMouseLeave}
        >
          <div className="flex h-full flex-col overflow-y-auto bg-white">
            <div>
              <div className="flex items-center gap-3 h-12 px-4">
                <img src={SuperplaneLogo} alt="Superplane" className="h-8 w-8" />
              </div>
            </div>

            <div className="p-4 border-b border-t border-gray-300">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-gray-500">Organization</p>
              <div className="mt-2 flex items-center gap-3">
                {/* <Avatar
                  initials={organizationInitial}
                  alt={organizationName}
                  className="size-8 bg-gray-900 text-gray-100 font-semibold"
                /> */}
                <div className="min-w-0">
                  <p className="font-semibold text-gray-900 truncate">{organizationName}</p>
                </div>
              </div>
              <div className="mt-2 flex flex-col">
                {sidebarOrganizationLinks.map((link) => (
                  <a
                    key={link.label}
                    href={link.href}
                    className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:bg-blue-100 hover:text-gray-900"
                  >
                    <Icon name={link.icon} size="sm" className="text-gray-500 transition group-hover:text-gray-900" />
                    <span>{link.label}</span>
                  </a>
                ))}
              </div>
            </div>

            <div className="p-4">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-gray-500">You</p>
              <div className="mt-2 flex items-center gap-3">
                {/* <Avatar
                  src={account?.avatar_url}
                  initials={accountInitials}
                  alt={account?.name || "User"}
                  className="size-8 bg-gray-900 text-gray-100"
                /> */}
                <div className="min-w-0">
                  <p className="font-semibold text-gray-900 truncate">{account?.name || "Loading..."}</p>
                  <p className="text-[13px] text-gray-500 font-medium truncate">{account?.email || "Loading..."}</p>
                </div>
              </div>
              <div className="mt-2 flex flex-col">
                {sidebarUserLinks.map((link) =>
                  link.href ? (
                    <a
                      key={link.label}
                      href={link.href}
                      className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:bg-blue-100 hover:text-gray-900"
                    >
                      <Icon name={link.icon} size="sm" className="text-gray-500 transition group-hover:text-gray-900" />
                      <span>{link.label}</span>
                    </a>
                  ) : (
                    <button
                      key={link.label}
                      type="button"
                      onClick={link.onClick}
                      className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-left text-sm font-medium text-gray-500 hover:bg-blue-100 hover:text-gray-900"
                    >
                      <Icon name={link.icon} size="sm" className="text-gray-500 transition group-hover:text-gray-900" />
                      <span>{link.label}</span>
                    </button>
                  ),
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
