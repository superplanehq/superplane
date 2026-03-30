import { BlobScopePanel } from "@/components/blobs/BlobScopePanel";

type BlobsSettingsProps = {
  organizationId: string;
};

export function BlobsSettings({ organizationId }: BlobsSettingsProps) {
  return <BlobScopePanel organizationId={organizationId} scopeType="SCOPE_TYPE_ORGANIZATION" />;
}
