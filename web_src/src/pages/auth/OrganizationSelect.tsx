import SuperplaneLogo from "@/assets/superplane.svg";
import { Building, Palette, User } from "lucide-react";
import React, { useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
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

  const createOrganizationCardClasses = createOrganizationDisabled
    ? "h-48 bg-slate-200/70 border-1 border-dashed border-slate-300 rounded-lg p-6 flex flex-col items-center justify-center text-slate-500 cursor-not-allowed"
    : "h-48 bg-white/50 hover:bg-white border-1 border-dashed border-gray-400 rounded-lg p-6 flex flex-col items-center justify-center transition-colors cursor-pointer";

  const createOrganizationCard = (
    <div
      className={createOrganizationCardClasses}
      aria-disabled={createOrganizationDisabled}
      tabIndex={createOrganizationDisabled ? 0 : -1}
    >
      <div className="flex items-center">
        <h4 className="text-lg font-medium text-center">+ Create new</h4>
      </div>
      <Text className="text-base/6 sm:text-sm/6 font-medium text-gray-500 text-center dark:text-gray-400">
        {createOrganizationDisabled
          ? "Organization limit reached for this account"
          : "Start fresh with a new organization"}
      </Text>
    </div>
  );

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-100 p-8">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500"></div>
          <Text className="text-gray-500 dark:text-gray-400">Loading...</Text>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-100">
      <div className="p-8">
        <div className="flex mb-4">
          <img src={SuperplaneLogo} alt="Superplane" className="h-6" />
        </div>
        <div className="max-w-7xl w-full">
          <div className="mb-6">
            <Text className="font-medium text-gray-800 text-left">
              Hey there{account?.name ? `, ${account.name}` : ""}!
            </Text>
            {organizations.length > 0 && (
              <Text className="font-medium text-gray-500 text-left">
                Select one of your organizations below to get started:
              </Text>
            )}
          </div>

          {error && (
            <div className="mb-6 p-3 rounded-md bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
              <Text className="text-red-700 text-sm">{error}</Text>
            </div>
          )}

          {organizations.length === 0 && (
            <div className="text-left py-2 mb-4">
              <Text className="font-medium text-gray-800">You're not a member of any organizations yet.</Text>
              <Text className="font-medium text-gray-800">
                {createOrganizationDisabled
                  ? "This account has reached its organization limit."
                  : "Create a new organization to get started!"}
              </Text>
            </div>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            {organizations.map((org) => (
              <Link
                key={org.id}
                to={`/${org.id}`}
                className="h-48 bg-white dark:bg-gray-900 rounded-md shadow-sm p-6 outline outline-slate-950/10 hover:outline-slate-950/20 hover:shadow-md transition-colors cursor-pointer"
              >
                <div className="flex flex-col h-full justify-between">
                  <div>
                    <div className="flex items-center gap-2 mb-1 text-gray-800 dark:text-white">
                      <Building size={16} />
                      <h4 className="text-lg font-medium truncate">{org.name}</h4>
                    </div>
                  </div>

                  <div className="mt-3 text-sm font-medium text-gray-500 dark:text-gray-400 flex flex-col gap-1">
                    <div className="flex items-center gap-1.5">
                      <Palette size={16} />
                      {formatCount(org.canvasCount ?? 0, "canvas")}
                    </div>
                    <div className="flex items-center gap-1.5">
                      <User size={16} />
                      {formatCount(org.memberCount ?? 0, "member")}
                    </div>
                  </div>
                </div>
              </Link>
            ))}

            {createOrganizationDisabled ? (
              <Tooltip>
                <TooltipTrigger asChild>{createOrganizationCard}</TooltipTrigger>
                <TooltipContent side="top" className="max-w-xs">
                  {createOrganizationTooltip}
                </TooltipContent>
              </Tooltip>
            ) : (
              <Link to="/create" className="block">
                {createOrganizationCard}
              </Link>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default OrganizationSelect;
