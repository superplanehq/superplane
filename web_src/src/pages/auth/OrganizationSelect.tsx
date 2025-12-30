import SuperplaneLogo from "@/assets/superplane.svg";
import { Building, Palette, User } from "lucide-react";
import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Text } from "../../components/Text/text";
import { useAccount } from "../../contexts/AccountContext";

interface Organization {
  id: string;
  name: string;
  description?: string;
  canvasCount?: number;
  memberCount?: number;
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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { account } = useAccount();
  const navigate = useNavigate();

  useEffect(() => {
    fetchOrganizations();
  }, [account]);

  const fetchOrganizations = async () => {
    if (!account) return;

    try {
      const orgsResponse = await fetch("/organizations", {
        credentials: "include",
      });

      if (orgsResponse.ok) {
        const organizations = await orgsResponse.json();
        setOrganizations(organizations);
      } else {
        setError("Failed to load organizations");
      }
    } catch {
      setError("Failed to load organizations");
    } finally {
      setLoading(false);
    }
  };

  const handleOrganizationSelect = (org: Organization) => {
    // Redirect to the organization's app route
    window.location.href = `/${org.id}`;
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-100 p-8">
        <div className="flex flex-col items-center space-y-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b border-gray-500"></div>
          <Text className="text-gray-600 dark:text-gray-400">Loading...</Text>
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
              <Text className="font-medium text-gray-800">Create a new organization to get started!</Text>
            </div>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            {organizations.map((org) => (
              <div
                key={org.id}
                className="h-48 bg-white dark:bg-gray-900 rounded-md shadow-sm p-6 outline outline-slate-950/10 hover:outline-slate-950/15 hover:shadow-md transition-colors cursor-pointer flex flex-col justify-between"
                onClick={() => handleOrganizationSelect(org)}
              >
                <div className="flex items-center gap-2 mb-2 text-gray-800 dark:text-white">
                  <Building size={16} />
                  <h4 className="text-lg font-semibold truncate">{org.name}</h4>
                </div>
                <Text className="text-sm text-left text-gray-600 dark:text-gray-400 truncate">{org.description}</Text>
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
            ))}

            <div
              className="h-48 bg-white/50 hover:bg-white border-1 border-dashed border-gray-400 rounded-lg p-6 flex flex-col items-center justify-center transition-colors cursor-pointer"
              onClick={() => navigate("/create")}
            >
              <div className="flex items-center">
                <h4 className="text-sm font-semibold text-gray-800 text-center">+ Create new</h4>
              </div>
              <Text className="text-[13px] text-gray-500 font-medium text-center">
                Start fresh with a new organization
              </Text>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default OrganizationSelect;
