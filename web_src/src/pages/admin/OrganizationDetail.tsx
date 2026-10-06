import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { OrganizationSpendingExplorer } from "@/pages/factories/pages/organizationSettings/spending-redesign/OrganizationSpendingExplorer";
import { ArrowLeft, Pin, PinOff } from "lucide-react";
import React, { useEffect, useState } from "react";
import { Link, useParams } from "react-router";

import { OrgCanvasesTable } from "./OrgCanvasesTable";
import { OrgExperimentalFeaturesTable } from "./OrgExperimentalFeaturesTable";
import { OrgIntegrationsTable } from "./OrgIntegrationsTable";
import { OrgLLMCreditSection } from "./OrgLLMCreditSection";
import { OrgOverviewPanel } from "./OrgOverviewPanel";
import { OrgUsersTable } from "./OrgUsersTable";
import { OrganizationVelocityPanel } from "./OrganizationVelocityPanel";
import { useAdminOrganizationSpendingReport } from "./useAdminOrganizationSpendingReport";

const ORGANIZATION_TABS = [
  "overview",
  "users",
  "automations",
  "connections",
  "features",
  "spending",
  "velocity",
  "credits",
] as const;

type OrganizationTab = (typeof ORGANIZATION_TABS)[number];

function isOrganizationTab(value: string): value is OrganizationTab {
  return ORGANIZATION_TABS.some((tab) => tab === value);
}

function OrganizationPinControl({ orgId }: { orgId: string }) {
  const [pinned, setPinned] = useState<boolean | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        const response = await fetch(`/admin/api/organizations/${orgId}`, { credentials: "include" });
        if (!response.ok) {
          return;
        }
        const data: { pinned?: boolean } = await response.json();
        if (!cancelled) {
          setPinned(Boolean(data.pinned));
        }
      } catch {
        if (!cancelled) {
          setPinned(null);
        }
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [orgId]);

  if (pinned === null) {
    return null;
  }

  const togglePin = async () => {
    setPending(true);
    setError(null);
    try {
      const response = await fetch(`/admin/api/organizations/${orgId}/pin`, {
        method: pinned ? "DELETE" : "PUT",
        credentials: "include",
      });
      if (!response.ok) {
        setError(pinned ? "Could not unpin this organization." : "Could not pin this organization.");
        return;
      }
      const data: { pinned?: boolean } = await response.json();
      setPinned(Boolean(data.pinned));
    } catch {
      setError(pinned ? "Could not unpin this organization." : "Could not pin this organization.");
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="flex flex-col items-end gap-1">
      <Button type="button" variant="outline" size="sm" disabled={pending} onClick={() => void togglePin()}>
        {pinned ? <PinOff /> : <Pin />}
        {pinned ? "Unpin" : "Pin"}
      </Button>
      {error ? <Text className="text-xs text-red-600 dark:text-red-400">{error}</Text> : null}
    </div>
  );
}

const OrganizationDetail: React.FC = () => {
  const { orgId } = useParams<{ orgId: string }>();
  const [tab, setTab] = useState<OrganizationTab>("overview");
  const [creditsVisited, setCreditsVisited] = useState(false);
  const [spendingVisited, setSpendingVisited] = useState(false);
  const [velocityVisited, setVelocityVisited] = useState(false);

  useReportPageReady(true);

  return (
    <div>
      <div className="mb-4 flex items-center justify-between gap-4">
        <Link
          to="/admin"
          className="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
        >
          <ArrowLeft size={14} />
          All organizations
        </Link>
        {orgId ? <OrganizationPinControl orgId={orgId} /> : null}
      </div>
      <Tabs
        value={tab}
        onValueChange={(nextTab) => {
          if (!isOrganizationTab(nextTab)) {
            return;
          }
          if (nextTab === "credits") {
            setCreditsVisited(true);
          }
          if (nextTab === "spending") {
            setSpendingVisited(true);
          }
          if (nextTab === "velocity") {
            setVelocityVisited(true);
          }
          setTab(nextTab);
        }}
      >
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
          <TabsTrigger value="automations">Automations</TabsTrigger>
          <TabsTrigger value="connections">Connections</TabsTrigger>
          <TabsTrigger value="features">Features</TabsTrigger>
          <TabsTrigger value="spending">Spending</TabsTrigger>
          <TabsTrigger value="velocity">Velocity</TabsTrigger>
          <TabsTrigger value="credits">Credits</TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="mt-3">
          <OrgOverviewPanel orgId={orgId!} />
        </TabsContent>
        <TabsContent value="users" className="mt-3">
          <OrgUsersTable orgId={orgId!} />
        </TabsContent>
        <TabsContent value="automations" className="mt-3">
          <OrgCanvasesTable orgId={orgId!} />
        </TabsContent>
        <TabsContent value="connections" className="mt-3">
          <OrgIntegrationsTable orgId={orgId!} />
        </TabsContent>
        <TabsContent value="features" className="mt-3">
          <OrgExperimentalFeaturesTable orgId={orgId!} />
        </TabsContent>
        <TabsContent
          value="spending"
          forceMount={spendingVisited || undefined}
          className="mt-3 data-[state=inactive]:hidden"
        >
          <OrganizationSpendingExplorer
            organizationId={orgId!}
            setDocumentTitle={false}
            useReport={useAdminOrganizationSpendingReport}
          />
        </TabsContent>
        <TabsContent
          value="velocity"
          forceMount={velocityVisited || undefined}
          className="mt-3 data-[state=inactive]:hidden"
        >
          <OrganizationVelocityPanel orgId={orgId!} />
        </TabsContent>
        <TabsContent
          value="credits"
          forceMount={creditsVisited || undefined}
          className="mt-3 data-[state=inactive]:hidden"
        >
          <OrgLLMCreditSection orgId={orgId!} />
        </TabsContent>
      </Tabs>
    </div>
  );
};

export default OrganizationDetail;
