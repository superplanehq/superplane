import SuperplaneLogo from "@/assets/superplane.svg";
import { useAccount } from "@/contexts/useAccount";
import { useOrganization, useOrganizationUsage } from "@/hooks/useOrganizationData";
import { isUsagePageForced } from "@/lib/env";
import { cn } from "@/lib/utils";
import {
  ArrowRightLeft,
  Bot,
  CircleUser,
  Gauge,
  Key,
  Lock,
  LogOut,
  Mail,
  Menu,
  Plug,
  Settings,
  Shield,
  User as UserIcon,
  Users,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/usePermissions";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/ui/tooltip";
import { Badge } from "@/components/ui/badge";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { posthog } from "@/posthog";

interface OrganizationMenuButtonProps {
  organizationId?: string;
  className?: string;
}

export function OrganizationMenuButton({ organizationId, className }: OrganizationMenuButtonProps) {
  const { account } = useAccount();
  const { data: organization } = useOrganization(organizationId || "");
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const canReadOrg = permissionsLoading || canAct("org", "read");
  const { data: usageStatus, error: usageError } = useOrganizationUsage(
    organizationId || "",
    !!organizationId && canReadOrg,
  );
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);

  const handleMenuButtonClick = () => {
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
  const usageEnabled = usageStatus?.enabled === true || !!usageError || isUsagePageForced();
  const logoHref = organizationId ? `/${organizationId}` : "/";

  const sidebarUserLinks = [
    ...(organizationId
      ? [
          {
            label: "Profile",
            href: `/${organizationId}/settings/profile`,
            Icon: CircleUser,
          },
        ]
      : []),
    ...(account?.installation_admin ? [{ label: "Installation Admin", href: "/admin", Icon: Shield }] : []),
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
    ...(usageEnabled
      ? [
          {
            label: "Usage",
            href: organizationId ? `/${organizationId}/settings/billing` : "#",
            Icon: Gauge,
            permission: { resource: "org", action: "read" },
          },
        ]
      : []),
    {
      label: "Secrets",
      href: organizationId ? `/${organizationId}/settings/secrets` : "#",
      Icon: Key,
      permission: { resource: "secrets", action: "read" },
    },
    { label: "Change Organization", href: "/?select=true", Icon: ArrowRightLeft },
  ];

  const handleSignOut = () => {
    setIsMenuOpen(false);
    posthog.reset();
    window.location.href = "/logout";
  };

  return (
    <TooltipProvider delayDuration={400}>
      <div className={cn("relative flex items-center", className)} ref={menuRef}>
        <div className="relative shrink-0">
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={handleMenuButtonClick}
                className="-ml-2 flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-gray-600 hover:bg-slate-100 hover:text-gray-900 cursor-pointer"
                aria-label="Open organization menu"
                aria-expanded={isMenuOpen}
                aria-haspopup="menu"
              >
                <Menu className="h-5 w-5" aria-hidden />
              </button>
            </TooltipTrigger>
            <TooltipContent>Open Menu</TooltipContent>
          </Tooltip>
          {isMenuOpen && (
            <div className="absolute -left-2 top-0 z-50 w-full min-w-[15rem] animate-in fade-in-0 slide-in-from-left-4 rounded-md border border-slate-950/20 bg-white shadow-md duration-200">
              {organizationId && (
                <div className="px-4 py-2 border-b border-gray-300">
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
                        !link.permission ||
                        permissionsLoading ||
                        canAct(link.permission.resource, link.permission.action);

                      if (!allowed) {
                        return (
                          <PermissionTooltip
                            key={link.label}
                            allowed={false}
                            message={`You don't have permission to view ${link.label.toLowerCase()}.`}
                            className="w-full"
                          >
                            <button
                              type="button"
                              className={cn(
                                "group flex items-center gap-2 rounded-md px-1.5 py-1 text-left text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800",
                                "opacity-60 cursor-not-allowed hover:bg-transparent hover:text-gray-500",
                              )}
                              disabled
                            >
                              <MenuIcon size={16} className="text-gray-500 transition group-hover:text-gray-800" />
                              <span>{link.label}</span>
                              <Lock size={12} className="ml-auto text-gray-400" />
                            </button>
                          </PermissionTooltip>
                        );
                      }

                      return link.href ? (
                        <Link
                          key={link.label}
                          to={link.href}
                          onClick={() => setIsMenuOpen(false)}
                          className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-left text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800"
                        >
                          <MenuIcon size={16} className="text-gray-500 transition group-hover:text-gray-800" />
                          <span>{link.label}</span>
                        </Link>
                      ) : (
                        <button
                          key={link.label}
                          type="button"
                          className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-left text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800"
                        >
                          <MenuIcon size={16} className="text-gray-500 transition group-hover:text-gray-800" />
                          <span>{link.label}</span>
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}
              <div className="px-4 py-2">
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
                      <Link
                        key={link.label}
                        to={link.href}
                        className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800"
                        onClick={() => setIsMenuOpen(false)}
                      >
                        <MenuIcon size={16} className="text-gray-500 transition group-hover:text-gray-800" />
                        <span>{link.label}</span>
                      </Link>
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
        <Tooltip>
          <TooltipTrigger asChild>
            <Link
              to={logoHref}
              aria-label="Go to canvases"
              className="flex h-8 cursor-pointer items-center rounded-md px-2 hover:bg-slate-100"
            >
              <img src={SuperplaneLogo} alt="SuperPlane" className="h-6 w-6" />
            </Link>
          </TooltipTrigger>
          <TooltipContent>Homepage</TooltipContent>
        </Tooltip>
        <HoverCard openDelay={100} closeDelay={150}>
          <HoverCardTrigger asChild>
            <Badge className="cursor-pointer rounded border-transparent bg-blue-500 px-1 py-0 text-[11px] font-semibold uppercase tracking-wide text-white hover:bg-blue-600">
              Beta
            </Badge>
          </HoverCardTrigger>
          <HoverCardContent align="start" className="w-72">
            <p className="text-sm font-semibold text-gray-800">We're just getting started!</p>
            <p className="mt-1 text-[13px] text-gray-500">SuperPlane is in beta. We'd love your feedback:</p>
            <div className="mt-3 flex flex-col">
              <a
                href="https://github.com/superplanehq/superplane/issues/new"
                target="_blank"
                rel="noopener noreferrer"
                className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800"
              >
                <IntegrationIcon integrationName="github" />
                <span>Log an issue</span>
              </a>
              <a
                href="https://discord.superplane.com/"
                target="_blank"
                rel="noopener noreferrer"
                className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800"
              >
                <IntegrationIcon integrationName="discord" />
                <span>Chat on Discord</span>
              </a>
              <a
                href="mailto:support@superplane.com"
                className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:bg-sky-100 hover:text-gray-800"
              >
                <Mail size={16} className="text-gray-500 transition group-hover:text-gray-800" />
                <span>Send us an email</span>
              </a>
            </div>
          </HoverCardContent>
        </HoverCard>
      </div>
    </TooltipProvider>
  );
}
