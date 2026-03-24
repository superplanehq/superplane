import { ArrowLeft } from "lucide-react";
import React from "react";
import { Link, useParams } from "react-router-dom";
import { OrgCanvasesTable } from "./OrgCanvasesTable";
import { OrgUsersTable } from "./OrgUsersTable";

const OrganizationDetail: React.FC = () => {
  const { orgId } = useParams<{ orgId: string }>();

  return (
    <div>
      <Link to="/admin" className="inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 mb-4">
        <ArrowLeft size={14} />
        All organizations
      </Link>
      <OrgUsersTable orgId={orgId!} />
      <OrgCanvasesTable orgId={orgId!} />
    </div>
  );
};

export default OrganizationDetail;
