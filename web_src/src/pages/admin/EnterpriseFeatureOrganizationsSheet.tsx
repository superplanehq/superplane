import { enterpriseFeaturesListEnterpriseFeatureOrganizations } from "@/api-client/sdk.gen";
import type { EnterpriseFeaturesEnterpriseFeatureOrganization } from "@/api-client/types.gen";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { enterpriseFeatureLabel } from "@/lib/license";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/ui/sheet";
import { useEffect, useState } from "react";
import { Link } from "react-router";

const sourceLabel = (source?: string) => {
  if (source === "auto_single_org") return "Enabled automatically";
  if (source === "admin") return "Enabled by an administrator";
  return "";
};

export function EnterpriseFeatureOrganizationsSheet({
  featureKey,
  onClose,
}: {
  featureKey: string;
  onClose: () => void;
}) {
  const [organizations, setOrganizations] = useState<EnterpriseFeaturesEnterpriseFeatureOrganization[]>([]);
  const [total, setTotal] = useState(0);
  const [pageToken, setPageToken] = useState("");
  const [previousTokens, setPreviousTokens] = useState<string[]>([]);
  const [nextPageToken, setNextPageToken] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError(false);
    enterpriseFeaturesListEnterpriseFeatureOrganizations({
      path: { featureKey },
      query: { pageSize: 50, pageToken: pageToken || undefined },
    })
      .then((response) => {
        if (!active) return;
        setOrganizations(response.data?.organizations ?? []);
        setTotal(Number(response.data?.total ?? 0));
        setNextPageToken(response.data?.nextPageToken ?? "");
      })
      .catch(() => {
        if (active) setError(true);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [featureKey, pageToken]);

  return (
    <Sheet
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <SheetContent
        overlayClassName="bg-black/20"
        side="right"
        className="flex h-full w-full flex-col gap-0 p-0 sm:w-[640px] sm:max-w-[640px]"
      >
        <SheetHeader className="shrink-0 border-b p-6 text-left">
          <SheetTitle>{enterpriseFeatureLabel(featureKey)}</SheetTitle>
          <Text className="text-sm text-gray-500 dark:text-gray-400">Organizations that can use this feature.</Text>
        </SheetHeader>
        <div className="min-h-0 flex-1 overflow-auto p-6">
          {loading ? <Text className="text-sm text-gray-500">Loading organizations...</Text> : null}
          {error ? <Text className="text-sm text-red-600">Failed to load organizations.</Text> : null}
          {!loading && !error && organizations.length === 0 ? (
            <Text className="text-sm text-gray-500">No organization uses this feature.</Text>
          ) : null}
          <ul className="space-y-3">
            {organizations.map((organization) => (
              <li key={organization.organizationId}>
                <Link
                  to={`/admin/organizations/${organization.organizationId}`}
                  className="font-medium text-gray-900 underline-offset-2 hover:underline dark:text-gray-100"
                >
                  {organization.name}
                </Link>
                <Text className="text-xs text-gray-500 dark:text-gray-400">
                  {organization.slug}
                  {sourceLabel(organization.source) ? ` · ${sourceLabel(organization.source)}` : ""}
                </Text>
              </li>
            ))}
          </ul>
        </div>
        <div className="flex items-center justify-between border-t p-4">
          <Text className="text-xs text-gray-500">{total} organizations</Text>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={previousTokens.length === 0}
              onClick={() => {
                const tokens = [...previousTokens];
                const previous = tokens.pop() ?? "";
                setPreviousTokens(tokens);
                setPageToken(previous);
              }}
            >
              Previous
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={!nextPageToken}
              onClick={() => {
                setPreviousTokens((tokens) => [...tokens, pageToken]);
                setPageToken(nextPageToken);
              }}
            >
              Next
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
