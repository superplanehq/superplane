import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { ArrowLeft } from "lucide-react";
import React, { useState } from "react";
import { Link, useParams } from "react-router";
import { OrgCanvasesTable } from "./OrgCanvasesTable";
import { OrgExperimentalFeaturesTable } from "./OrgExperimentalFeaturesTable";
import { OrgLLMCreditSection } from "./OrgLLMCreditSection";
import { OrgOverviewPanel } from "./OrgOverviewPanel";
import { OrgUsersTable } from "./OrgUsersTable";

const ORGANIZATION_TABS = ["overview", "users", "automations", "features", "credits"] as const;

type OrganizationTab = (typeof ORGANIZATION_TABS)[number];

function isOrganizationTab(value: string): value is OrganizationTab {
  return ORGANIZATION_TABS.some((tab) => tab === value);
}

const OrganizationDetail: React.FC = () => {
  const { orgId } = useParams<{ orgId: string }>();
  const [tab, setTab] = useState<OrganizationTab>("overview");
  const [creditsVisited, setCreditsVisited] = useState(false);

  useReportPageReady(true);

  return (
    <div>
      <Link
        to="/admin"
        className="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 mb-4 dark:text-gray-400 dark:hover:text-gray-200"
      >
        <ArrowLeft size={14} />
        All organizations
      </Link>
      <Tabs
        value={tab}
        onValueChange={(nextTab) => {
          if (!isOrganizationTab(nextTab)) {
            return;
          }
          if (nextTab === "credits") {
            setCreditsVisited(true);
          }
          setTab(nextTab);
        }}
      >
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
          <TabsTrigger value="automations">Automations</TabsTrigger>
          <TabsTrigger value="features">Features</TabsTrigger>
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
        <TabsContent value="features" className="mt-3">
          <OrgExperimentalFeaturesTable orgId={orgId!} />
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
