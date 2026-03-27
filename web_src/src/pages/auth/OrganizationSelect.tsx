import { Heading } from "@/components/Heading/heading";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Palette, Plus, User } from "lucide-react";
import React, { useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { getUsageLimitNotice } from "@/utils/usageLimits";
import { Text } from "../../components/Text/text";
import { useAccount } from "../../contexts/AccountContext";

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
    canvas: "canvases",
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
  }, [account, accountLoading, location.pathname, location.search, navigate]);

  const fetchOrganizations = async () => {
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
        const organizations = await orgsResponseResult.value.json();
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
  };

  const createOrganizationDisabled = organizationCreationStatus?.allowed === false;
  const createOrganizationTooltip =
    (createOrganizationDisabled ? getUsageLimitNotice(organizationCreationStatus?.message)?.description : null) ||
    organizationCreationStatus?.message ||
    "This account cannot create another organization right now.";

  const listRowMinHeight = "min-h-[58px]";

  const createOrganizationEnabledClasses = cn(
    "relative flex w-full flex-row items-center gap-4 rounded-md border border-dashed border-green-500 bg-green-50 px-4 py-3 transition-colors dark:border-green-500 dark:bg-green-950/30 hover:bg-green-100 dark:hover:bg-green-950/50 cursor-pointer",
    listRowMinHeight,
  );

  const createOrganizationDisabledClasses = cn(
    "relative flex w-full flex-row items-center gap-4 rounded-md border border-dashed border-slate-300 bg-slate-200/70 px-4 py-3 text-slate-500 cursor-not-allowed dark:border-slate-600 dark:bg-slate-800/50",
    listRowMinHeight,
  );

  const createOrganizationInner = (
    <>
      <span
        className={
          createOrganizationDisabled
            ? "flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-slate-400 text-white dark:bg-slate-500"
            : "flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-green-500 text-white"
        }
      >
        <Plus className="h-4 w-4" strokeWidth={2} aria-hidden />
      </span>
      <Heading
        level={3}
        className="!text-base font-medium text-gray-800 transition-colors mb-0 !leading-6 line-clamp-2 truncate dark:text-gray-100"
      >
        <span className="truncate">New Organization</span>
      </Heading>
    </>
  );

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-100">
        <div className="px-4 py-2 bg-white border-b border-slate-200">
          <OrganizationMenuButton />
        </div>
        <div className="p-8 flex justify-center">
          <div className="w-full max-w-[640px] flex flex-col items-center justify-center gap-4 py-16">
            <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500"></div>
            <Text className="text-gray-500 dark:text-gray-400">Loading...</Text>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-100">
      <div className="px-4 py-2 bg-white border-b border-slate-200">
        <OrganizationMenuButton />
      </div>
      <div className="p-8 flex justify-center">
        <div className="w-full max-w-[640px] mx-auto">
          <div className="flex flex-col items-start mb-6">
            <div className="w-full text-left">
              <Text className="font-medium text-gray-800 block">
                Hey there{account?.name ? `, ${account.name}` : ""}!
              </Text>
              {organizations.length > 0 && (
                <Text className="font-medium text-gray-500 block dark:text-gray-400">
                  Select one of your organizations below to get started:
                </Text>
              )}
            </div>
          </div>

          {error && (
            <div className="mb-6 p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
              <Text className="text-red-700 text-sm">{error}</Text>
            </div>
          )}

          {organizations.length === 0 && (
            <div className="text-left py-2 mb-4 space-y-1">
              <Text className="font-medium text-gray-800 block">You're not a member of any organizations yet.</Text>
              <Text className="font-medium text-gray-800 block">
                {createOrganizationDisabled
                  ? "This account has reached its organization limit."
                  : "Create a new organization to get started!"}
              </Text>
            </div>
          )}

          <ul className="flex flex-col gap-3 list-none p-0 m-0">
            {organizations.map((org) => (
              <li key={org.id}>
                <Link
                  to={`/${org.id}`}
                  className={cn(
                    "flex items-center justify-between gap-4 rounded-md bg-white dark:bg-gray-900 px-4 py-3 shadow-sm outline outline-slate-950/10 hover:outline-slate-950/20 hover:shadow-md transition-colors cursor-pointer",
                    listRowMinHeight,
                  )}
                >
                  <div className="flex items-center gap-4 min-w-0">
                    <span
                      className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-slate-800 text-sm font-medium text-white"
                      aria-hidden
                    >
                      {organizationInitial(org.name)}
                    </span>
                    <span className="text-base font-medium text-gray-800 dark:text-white truncate">{org.name}</span>
                  </div>

                  <div className="flex items-center gap-3 sm:gap-4 shrink-0 text-xs sm:text-sm font-medium text-gray-500 dark:text-gray-400">
                    <span className="flex items-center gap-1.5 whitespace-nowrap">
                      <Palette size={14} className="shrink-0" aria-hidden />
                      {formatCount(org.canvasCount ?? 0, "canvas")}
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
