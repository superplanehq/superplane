import { useEffect, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { useOrganization, useUpdateOrganization } from "@/hooks/useOrganizationData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { organizationSlugValidationMessage, validateOrganizationSlug } from "@/lib/organizationSlug";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { FactorySettingsCard, FactorySettingsPageFrame } from "../settings/FactorySettingsCard";

export function OrganizationSettingsOverviewPage() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: organization } = useOrganization(organizationId || "");
  const updateOrganizationMutation = useUpdateOrganization(organizationId || "");
  const organizationName = organization?.metadata?.name || "Organization";
  const currentSlug = organization?.metadata?.slug || "";

  const [slug, setSlug] = useState(currentSlug);
  const [slugError, setSlugError] = useState<string | null>(null);

  useEffect(() => {
    setSlug(currentSlug);
    setSlugError(null);
  }, [currentSlug]);

  usePageTitle(["General", organizationName]);

  const canUpdateOrg = canAct("org", "update");
  const trimmedSlug = slug.trim();
  const slugChanged = trimmedSlug !== currentSlug;

  const handleSave = async () => {
    if (!canUpdateOrg || !organizationId) return;

    if (!slugChanged) return;

    const validationError = validateOrganizationSlug(trimmedSlug);
    if (validationError) {
      setSlugError(organizationSlugValidationMessage(validationError));
      return;
    }
    setSlugError(null);

    try {
      await updateOrganizationMutation.mutateAsync({
        name: organization?.metadata?.name,
        slug: trimmedSlug,
      });

      showSuccessToast("Organization updated.");

      // The URL segment only needs rewriting when it currently carries the
      // old slug. When it carries the organization ID instead, leave it
      // alone so navigation keeps working.
      if (organizationId === currentSlug) {
        const newPath = location.pathname.replace(`/${organizationId}/`, `/${trimmedSlug}/`);
        navigate(`${newPath}${location.search}`, { replace: true });
      }
    } catch (err) {
      const message = getApiErrorMessage(err, "Failed to update organization slug");
      setSlugError(message);
      showErrorToast(message);
    }
  };

  return (
    <FactorySettingsPageFrame title="General" subtitle="See the organization name and basic details.">
      <FactorySettingsCard title="Organization" data-testid="organization-settings-overview">
        <dl className="space-y-1">
          <dt className="text-[12px] text-muted-foreground">Name</dt>
          <dd className="text-[13px] text-foreground" data-testid="organization-settings-overview-name">
            {organizationName}
          </dd>
        </dl>

        <div className="mt-4 space-y-2">
          <Label htmlFor="organization-settings-overview-slug">Organization slug</Label>
          <Input
            id="organization-settings-overview-slug"
            data-testid="organization-settings-overview-slug-input"
            value={slug}
            onChange={(event) => {
              setSlug(event.target.value);
              setSlugError(null);
            }}
            className="max-w-sm"
            disabled={!canUpdateOrg}
          />
          <p className="text-[11px] text-muted-foreground">
            Used in your workspace URL. Use lowercase letters, numbers, and dashes only.
          </p>
          {slugError ? <p className="text-[11px] text-destructive">{slugError}</p> : null}
        </div>

        <div className="mt-4">
          <PermissionTooltip
            allowed={canUpdateOrg || permissionsLoading}
            message="You don't have permission to update this organization."
          >
            <LoadingButton
              type="button"
              data-testid="organization-settings-overview-save"
              onClick={() => void handleSave()}
              disabled={!canUpdateOrg || !slugChanged}
              loading={updateOrganizationMutation.isPending}
              loadingText="Saving..."
            >
              Save
            </LoadingButton>
          </PermissionTooltip>
        </div>
      </FactorySettingsCard>
    </FactorySettingsPageFrame>
  );
}
