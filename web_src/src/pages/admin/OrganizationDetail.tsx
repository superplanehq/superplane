import { ArrowLeft } from "lucide-react";
import React from "react";
import { Link, useParams } from "react-router-dom";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { OrgCanvasesTable } from "./OrgCanvasesTable";
import { OrgExperimentalFeaturesTable } from "./OrgExperimentalFeaturesTable";
import { OrgUsersTable } from "./OrgUsersTable";

const OrganizationDetail: React.FC = () => {
  const { orgId } = useParams<{ orgId: string }>();

  useReportPageReady(true);

  return (
    <div>
      <Link
        to="/admin"
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-content-secondary hover:text-content-primary"
      >
        <ArrowLeft size={14} />
        All organizations
      </Link>
      <OrgUsersTable orgId={orgId!} />
      <OrgCanvasesTable orgId={orgId!} />
      <OrgExperimentalFeaturesTable orgId={orgId!} />
    </div>
  );
};

export default OrganizationDetail;
