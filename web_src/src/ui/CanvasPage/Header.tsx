import SuperplaneLogo from "@/assets/superplane.svg";
// import { Avatar } from "@/components/Avatar/avatar";
import { Icon } from "@/components/Icon";
import { useAccount } from "@/contexts/AccountContext";
import { useOrganization } from "@/hooks/useOrganizationData";
import { resolveIcon } from "@/lib/utils";
import { ChevronDown, Undo2 } from "lucide-react";
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
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);

  const handleLogoButtonClick = () => {
    if (!organizationId) {
      onLogoClick?.();
      return;
    }

    setIsMenuOpen((prev) => !prev);
  };

  useEffect(() => {
    if (!isMenuOpen) return;

    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsMenuOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [isMenuOpen]);

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
    setIsMenuOpen(false);
    window.location.href = "/logout";
  };

  return (
    <>
      <header className="bg-white border-b border-border">
        <div className="relative flex items-center justify-between h-12 px-3">
          {/* Logo */}
          <div className="relative flex items-center" ref={menuRef}>
            <button
              onClick={handleLogoButtonClick}
              className="flex items-center gap-1 cursor-pointer"
              aria-label="Open organization menu"
              aria-expanded={isMenuOpen}
            >
              <img src={SuperplaneLogo} alt="Logo" className="w-7 h-7" />
              {organizationId && (
                <ChevronDown
                  size={16}
                  className={`text-gray-400 transition-transform ${isMenuOpen ? "rotate-180" : ""}`}
                />
              )}
            </button>
            {organizationId && isMenuOpen && (
              <div className="absolute left-0 top-13 z-50 w-60 rounded-md outline outline-slate-950/15 bg-white shadow-lg">
                <div className="px-4 pt-3 pb-4 border-b border-gray-300">
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-gray-100 bg-gray-800 inline px-1 py-0.5 rounded">
                    Org
                  </p>
                  <div className="flex items-center gap-3 mt-2">
                    <div className="min-w-0">
                      <p className="font-semibold text-gray-800 truncate text-sm">{organizationName}</p>
                    </div>
                  </div>
                  <div className="mt-2 flex flex-col">
                    {sidebarOrganizationLinks.map((link) => (
                      <a
                        key={link.label}
                        href={link.href}
                        className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800"
                        onClick={() => setIsMenuOpen(false)}
                      >
                        <Icon
                          name={link.icon}
                          size="sm"
                          className="text-gray-500 transition group-hover:text-gray-800"
                        />
                        <span>{link.label}</span>
                      </a>
                    ))}
                  </div>
                </div>
                <div className="px-4 pt-3 pb-4">
                  <p className="text-[11px] font-semibold uppercase tracking-wide text-purple-600 bg-purple-200 inline px-1 py-0.5 rounded">
                    You
                  </p>
                  <div className="flex items-center gap-3 mt-2">
                    <div className="min-w-0">
                      <p className="font-semibold text-gray-800 truncate text-sm">{account?.name || "Loading..."}</p>
                      <p className="text-[13px] text-gray-500 font-medium truncate">{account?.email || "Loading..."}</p>
                    </div>
                  </div>
                  <div className="mt-2 flex flex-col">
                    {sidebarUserLinks.map((link) =>
                      link.href ? (
                        <a
                          key={link.label}
                          href={link.href}
                          className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800"
                          onClick={() => setIsMenuOpen(false)}
                        >
                          <Icon
                            name={link.icon}
                            size="sm"
                            className="text-gray-500 transition group-hover:text-gray-800"
                          />
                          <span>{link.label}</span>
                        </a>
                      ) : (
                        <button
                          key={link.label}
                          type="button"
                          onClick={link.onClick}
                          className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-left text-sm font-medium text-gray-500 hover:bg-blue-100 hover:text-gray-800"
                        >
                          <Icon
                            name={link.icon}
                            size="sm"
                            className="text-gray-500 transition group-hover:text-gray-800"
                          />
                          <span>{link.label}</span>
                        </button>
                      ),
                    )}
                  </div>
                </div>
              </div>
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
    </>
  );
}
