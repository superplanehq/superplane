import SuperplaneLogo from "@/assets/superplane.svg";
import { useAccount } from "@/contexts/AccountContext";
import { useOrganization } from "@/hooks/useOrganizationData";
import { cn } from "@/lib/utils";
import {
  ArrowRightLeft,
  Bot,
  ChevronDown,
  CircleUser,
  Key,
  Lock,
  LogOut,
  Plug,
  Settings,
  Shield,
  User as UserIcon,
  Users,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/PermissionsContext";

interface OrganizationMenuButtonProps {
  organizationId?: string;
  onLogoClick?: () => void;
  className?: string;
}

export function OrganizationMenuButton({ organizationId, onLogoClick, className }: OrganizationMenuButtonProps) {
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId || "");
  const { canAct, isLoading: permissionsLoading } = usePermissions();
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

    const handlePointerDown = (event: MouseEvent | TouchEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsMenuOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsMenuOpen(false);
      }
    };

    const listenerOptions: AddEventListenerOptions = { capture: true };

    document.addEventListener("mousedown", handlePointerDown, listenerOptions);
    document.addEventListener("touchstart", handlePointerDown, listenerOptions);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown, listenerOptions);
      document.removeEventListener("touchstart", handlePointerDown, listenerOptions);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isMenuOpen]);

  const organizationName = organization?.metadata?.name || "Organization";

  const sidebarUserLinks = [
    {
      label: "Profile",
      href: organizationId ? `/${organizationId}/settings/profile` : "#",
      Icon: CircleUser,
    },
    {
      label: "Sign Out",
      Icon: LogOut,
      onClick: () => handleSignOut(),
    },
  ];

  const sidebarOrganizationLinks = [
    {
      label: "Settings",
      href: organizationId ? `/${organizationId}/settings/general` : "#",
      Icon: Settings,
      permission: { resource: "org", action: "read" },
    },
    {
      label: "Members",
      href: organizationId ? `/${organizationId}/settings/members` : "#",
      Icon: UserIcon,
      permission: { resource: "members", action: "read" },
    },
    {
      label: "Service Accounts",
      href: organizationId ? `/${organizationId}/settings/service-accounts` : "#",
      Icon: Bot,
      permission: { resource: "service_accounts", action: "read" },
    },
    {
      label: "Groups",
      href: organizationId ? `/${organizationId}/settings/groups` : "#",
      Icon: Users,
      permission: { resource: "groups", action: "read" },
    },
    {
      label: "Roles",
      href: organizationId ? `/${organizationId}/settings/roles` : "#",
      Icon: Shield,
      permission: { resource: "roles", action: "read" },
    },
    {
      label: "Integrations",
      href: organizationId ? `/${organizationId}/settings/integrations` : "#",
      Icon: Plug,
      permission: { resource: "integrations", action: "read" },
    },
    {
      label: "Secrets",
      href: organizationId ? `/${organizationId}/settings/secrets` : "#",
      Icon: Key,
      permission: { resource: "secrets", action: "read" },
    },
    { label: "Change Organization", href: "/", Icon: ArrowRightLeft },
  ];

  const handleSignOut = () => {
    setIsMenuOpen(false);
    window.location.href = "/logout";
  };

  return (
    <div className={cn("relative flex items-center", className)} ref={menuRef}>
      <button
        onClick={handleLogoButtonClick}
        className="flex items-center gap-1 cursor-pointer"
        aria-label="Open organization menu"
        aria-expanded={isMenuOpen}
      >
        <img src={SuperplaneLogo} alt="Logo" className="w-7 h-7" />
        {organizationId && (
          <ChevronDown size={16} className={`text-gray-400 transition-transform ${isMenuOpen ? "rotate-180" : ""}`} />
        )}
      </button>
      {organizationId && isMenuOpen && (
        <div className="absolute left-0 top-13 z-50 w-60 rounded-md outline outline-slate-950/20 bg-white shadow-lg">
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
              {sidebarOrganizationLinks.map((link) => {
                const MenuIcon = link.Icon;
                const allowed =
                  !link.permission || permissionsLoading || canAct(link.permission.resource, link.permission.action);
                const linkButton = (
                  <button
                    key={link.label}
                    type="button"
                    onClick={() => {
                      if (!allowed) return;
                      setIsMenuOpen(false);
                      if (link.href) {
                        window.location.href = link.href;
                      }
                    }}
                    className={cn(
                      "group flex items-center gap-2 rounded-md px-1.5 py-1 text-left text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800",
                      !allowed && "opacity-60 cursor-not-allowed hover:bg-transparent hover:text-gray-500",
                    )}
                    disabled={!allowed}
                  >
                    <MenuIcon size={16} className="text-gray-500 transition group-hover:text-gray-800" />
                    <span>{link.label}</span>
                    {!allowed && <Lock size={12} className="ml-auto text-gray-400" />}
                  </button>
                );

                if (allowed) {
                  return linkButton;
                }

                return (
                  <PermissionTooltip
                    key={link.label}
                    allowed={false}
                    message={`You don't have permission to view ${link.label.toLowerCase()}.`}
                    className="w-full"
                  >
                    {linkButton}
                  </PermissionTooltip>
                );
              })}
            </div>
          </div>
          <div className="px-4 pt-3 pb-4">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-white bg-sky-500 inline px-1 py-0.5 rounded">
              You
            </p>
            <div className="flex items-center gap-3 mt-2">
              <div className="min-w-0">
                <p className="font-semibold text-gray-800 truncate text-sm">{account?.name || "Loading..."}</p>
                <p className="text-[13px] text-gray-500 font-medium truncate">{account?.email || "Loading..."}</p>
              </div>
            </div>
            <div className="mt-2 flex flex-col">
              {sidebarUserLinks.map((link) => {
                const MenuIcon = link.Icon;
                return link.href ? (
                  <a
                    key={link.label}
                    href={link.href}
                    className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800"
                    onClick={() => setIsMenuOpen(false)}
                  >
                    <MenuIcon size={16} className="text-gray-500 transition group-hover:text-gray-800" />
                    <span>{link.label}</span>
                  </a>
                ) : (
                  <button
                    key={link.label}
                    type="button"
                    onClick={link.onClick}
                    className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-left text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800"
                  >
                    <MenuIcon size={16} className="text-gray-500 transition group-hover:text-gray-800" />
                    <span>{link.label}</span>
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
