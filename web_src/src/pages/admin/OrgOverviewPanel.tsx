import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Building } from "lucide-react";
import { useEffect, useState } from "react";
import { formatDate } from "./formatDate";

interface OrganizationOverview {
  id: string;
  name: string;
  slug: string;
  description: string;
  canvas_count: number;
  member_count: number;
  created_at?: string;
  updated_at?: string;
}

function displayText(value: string): string {
  if (value.trim() === "") {
    return "—";
  }
  return value;
}

function OverviewField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <Text className="text-xs text-gray-500 dark:text-gray-400">{label}</Text>
      <Text className="mt-1 text-sm text-gray-800 dark:text-gray-100">{value}</Text>
    </div>
  );
}

export function OrgOverviewPanel({ orgId }: { orgId: string }) {
  const [organization, setOrganization] = useState<OrganizationOverview | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const response = await fetch(`/admin/api/organizations/${orgId}`, { credentials: "include" });
        if (!response.ok) {
          throw new Error("Could not load this organization.");
        }
        const data: OrganizationOverview = await response.json();
        if (!cancelled) {
          setOrganization(data);
        }
      } catch {
        if (!cancelled) {
          setOrganization(null);
          setError("Could not load this organization.");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [orgId]);

  return (
    <div className="mb-8">
      <div className="flex items-center gap-2 mb-3">
        <Building size={16} className="text-gray-600 dark:text-gray-400" />
        <Heading level={2} className="text-gray-800 text-base dark:text-gray-100">
          Overview
        </Heading>
      </div>
      {loading ? (
        <Text className="text-gray-500 text-sm dark:text-gray-400">Loading organization...</Text>
      ) : error ? (
        <Text className="text-red-600 text-sm dark:text-red-400">{error}</Text>
      ) : organization ? (
        <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 p-4 dark:bg-gray-900 dark:outline-gray-700/70">
          <div className="grid gap-4 sm:grid-cols-2">
            <OverviewField label="Name" value={organization.name} />
            <OverviewField label="Slug" value={displayText(organization.slug)} />
            <OverviewField label="Organization ID" value={organization.id} />
            <OverviewField label="Description" value={displayText(organization.description)} />
            <OverviewField label="Created" value={formatDate(organization.created_at)} />
            <OverviewField label="Updated" value={formatDate(organization.updated_at)} />
            <OverviewField label="Members" value={String(organization.member_count)} />
            <OverviewField label="Automations" value={String(organization.canvas_count)} />
          </div>
        </div>
      ) : null}
    </div>
  );
}
