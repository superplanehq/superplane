import { Heading } from "@/components/Heading/heading";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { LayoutPanelLeft, Plus, User } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import {
  createActionCardClassName,
  createActionCardDisabledClassName,
  createActionIconClassName,
  createActionIconDisabledClassName,
} from "@/lib/createActionStyles";
import { pickAutoRedirectOrganization, readLastVisitedOrganization } from "@/lib/lastVisitedOrganization";
import { getUsageLimitNotice } from "@/lib/usageLimits";
import { Text } from "../../components/Text/text";
import { useAccount } from "../../contexts/useAccount";
import { useReportPageReady } from "@/hooks/useReportPageReady";

interface Organization {
  id: string;
  name: string;
  canvasCount?: number;
  memberCount?: number;
}

interface OrganizationCreationStatus {
  allowed: boolean;
  usageEnabled: boolean;
  currentOrganizations: number;
  maxOrganizations: number;
  message?: string;
}

const organizationInitial = (name: string) => {
  const letter = name.trim().charAt(0);
  return letter ? letter.toUpperCase() : "?";
};

const formatCount = (count: number, noun: string) => {
  const safeCount = Number.isFinite(count) ? count : 0;
  const pluralOverrides: Record<string, string> = {
    member: "members",
  };
  const nounToUse = safeCount === 1 ? noun : pluralOverrides[noun] || `${noun}s`;
  return `${safeCount} ${nounToUse}`;
};
const OrganizationSelect: React.FC = () => {
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [organizationCreationStatus, setOrganizationCreationStatus] = useState<OrganizationCreationStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { account, loading: accountLoading } = useAccount();
  const navigate = useNavigate();
  const location = useLocation();

  const fetchOrganizations = useCallback(async () => {
    if (!account) {
      setLoading(false);
      return;
    }

    try {
      const [orgsResponseResult, creationStatusResponseResult] = await Promise.allSettled([
        fetch("/organizations", {
          credentials: "include",
        }),
        fetch("/account/limits", {
          credentials: "include",
        }),
      ]);

      if (orgsResponseResult.status === "fulfilled" && orgsResponseResult.value.ok) {
        const organizations = (await orgsResponseResult.value.json()) as Organization[];

        const explicitSelection = new URLSearchParams(location.search).has("select");
        if (!explicitSelection) {
          const redirectOrganizationId = pickAutoRedirectOrganization(
            organizations,
            readLastVisitedOrganization(account.id),
          );
          if (redirectOrganizationId) {
            navigate(`/${redirectOrganizationId}`, { replace: true });
            return;
          }
        }

        setOrganizations(organizations);
      } else {
        setError("Failed to load organizations");
      }

      if (creationStatusResponseResult.status === "fulfilled" && creationStatusResponseResult.value.ok) {
        const creationStatus = (await creationStatusResponseResult.value.json()) as OrganizationCreationStatus;
        setOrganizationCreationStatus(creationStatus);
      } else {
        setOrganizationCreationStatus(null);
      }
    } catch {
      setError("Failed to load organizations");
      setOrganizationCreationStatus(null);
    } finally {
      setLoading(false);
    }
  }, [account, location.search, navigate]);

  useEffect(() => {
    if (accountLoading) {
      return;
    }

    if (!account) {
      const redirectParam = encodeURIComponent(`${location.pathname}${location.search}`);
      navigate(`/login?redirect=${redirectParam}`, { replace: true });
      setLoading(false);
      return;
    }

    fetchOrganizations();
  }, [account, accountLoading, location.pathname, location.search, navigate, fetchOrganizations]);

  useReportPageReady(!loading && !accountLoading, {
    organization_count: organizations.length,
    failed: !!error,
  });

  const createOrganizationDisabled = organizationCreationStatus?.allowed === false;
  const createOrganizationTooltip =
    (createOrganizationDisabled ? getUsageLimitNotice(organizationCreationStatus?.message)?.description : null) ||
    organizationCreationStatus?.message ||
    "This account cannot create another organization right now.";

  const listRowMinHeight = "min-h-[58px]";

  const pageShellClassName = "flex min-h-screen flex-col bg-surface-canvas";

  const pageHeaderClassName = "flex h-12 items-center border-b border-edge-default bg-surface-default px-4";

  const organizationRowClassName = cn(
    "flex cursor-pointer items-center justify-between gap-4 rounded-md border border-edge-default bg-surface-raised px-4 py-3 shadow-sm transition-colors hover:bg-surface-subtle hover:shadow-md",
    listRowMinHeight,
  );

  const createOrganizationEnabledClasses = cn(createActionCardClassName, "cursor-pointer", listRowMinHeight);

  const createOrganizationDisabledClasses = cn(createActionCardDisabledClassName, listRowMinHeight);

  const createOrganizationInner = (
    <>
      <span className={createOrganizationDisabled ? createActionIconDisabledClassName : createActionIconClassName}>
        <Plus className="h-4 w-4" strokeWidth={2} aria-hidden />
      </span>
      <Heading
        level={3}
        className="mb-0 line-clamp-2 truncate !text-base !leading-6 font-medium text-content-primary transition-colors"
      >
        <span className="truncate">New Organization</span>
      </Heading>
    </>
  );

  if (loading) {
    return (
      <div className={pageShellClassName}>
        <header className={pageHeaderClassName}>
          <OrganizationMenuButton />
        </header>
        <div className="p-8 flex justify-center">
          <div className="w-full max-w-[640px] flex flex-col items-center justify-center gap-4 py-16">
            <div className="h-8 w-8 animate-spin rounded-full border-b border-focus-ring"></div>
            <Text className="text-content-secondary">Loading...</Text>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={pageShellClassName}>
      <header className={pageHeaderClassName}>
        <OrganizationMenuButton />
      </header>
      <div className="p-8 flex justify-center">
        <div className="w-full max-w-[640px] mx-auto">
          <div className="flex flex-col items-start mb-6">
            <div className="w-full text-left">
              <Text className="block font-medium text-content-primary">
                Hey there{account?.name ? `, ${account.name}` : ""}!
              </Text>
              {organizations.length > 0 && (
                <Text className="block font-medium text-content-secondary">
                  Select one of your organizations below to get started:
                </Text>
              )}
            </div>
          </div>

          {error && (
            <div className="mb-6 p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
              <Text className="text-red-700 dark:text-red-400 text-sm">{error}</Text>
            </div>
          )}

          {organizations.length === 0 && (
            <div className="text-left py-2 mb-4 space-y-1">
              <Text className="block font-medium text-content-primary">
                You're not a member of any organizations yet.
              </Text>
              <Text className="block font-medium text-content-secondary">
                {createOrganizationDisabled
                  ? "This account has reached its organization limit."
                  : "Create a new organization to get started!"}
              </Text>
            </div>
          )}

          <ul className="flex flex-col gap-3 list-none p-0 m-0">
            {organizations.map((org) => (
              <li key={org.id}>
                <Link to={`/${org.id}`} className={organizationRowClassName}>
                  <div className="flex items-center gap-4 min-w-0">
                    <span
                      className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-action-primary text-sm font-medium text-action-primary-content"
                      aria-hidden
                    >
                      {organizationInitial(org.name)}
                    </span>
                    <span className="truncate text-base font-medium text-content-primary">{org.name}</span>
                  </div>

                  <div className="flex shrink-0 items-center gap-3 text-xs font-medium text-content-secondary sm:gap-4 sm:text-sm">
                    <span className="flex items-center gap-1.5 whitespace-nowrap">
                      <LayoutPanelLeft size={14} className="shrink-0" aria-hidden />
                      {formatCount(org.canvasCount ?? 0, "app")}
                    </span>
                    <span className="flex items-center gap-1.5 whitespace-nowrap">
                      <User size={14} className="shrink-0" aria-hidden />
                      {formatCount(org.memberCount ?? 0, "member")}
                    </span>
                  </div>
                </Link>
              </li>
            ))}

            <li>
              {createOrganizationDisabled ? (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <div className={createOrganizationDisabledClasses} aria-disabled tabIndex={0}>
                      {createOrganizationInner}
                    </div>
                  </TooltipTrigger>
                  <TooltipContent side="top" className="max-w-xs">
                    {createOrganizationTooltip}
                  </TooltipContent>
                </Tooltip>
              ) : (
                <Link to="/create" className={createOrganizationEnabledClasses} aria-label="Create new organization">
                  {createOrganizationInner}
                </Link>
              )}
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export default OrganizationSelect;
