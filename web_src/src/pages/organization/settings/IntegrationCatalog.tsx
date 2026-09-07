import { Loader2, Plug, Search, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { PermissionTooltip } from "@/components/PermissionGate";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { getApiErrorMessage } from "@/lib/errors";
import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationInstructions } from "@/ui/IntegrationInstructions";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { Icon } from "@/components/Icon";
import { cn } from "@/lib/utils";
import { settingsModalClassName } from "./settingsPageStyles";
import { GitHubConnectControls } from "./GitHubConnectControls";
import { usesHostedGitHubAppInstall } from "@/lib/integrations";
import { useIntegrationCatalog } from "@/hooks/useIntegrationCatalog";
import { integrationStatusLabel, type IntegrationCatalogItem } from "@/lib/integrationCatalog";
import {
  catalogAppearance,
  instancePlugClass,
  instanceStatusLabelClass,
  type CatalogAppearance,
} from "./integrationCatalogAppearance";

type CatalogState = ReturnType<typeof useIntegrationCatalog>;
type CatalogStyles = ReturnType<typeof catalogAppearance>;

export function IntegrationCatalog({
  organizationId,
  appearance,
}: {
  organizationId: string;
  appearance: CatalogAppearance;
}) {
  const catalog = useIntegrationCatalog(organizationId);
  const styles = catalogAppearance(appearance);

  if (catalog.isLoading) {
    return <CatalogLoading appearance={appearance} />;
  }

  return (
    <div className={styles.root}>
      <CatalogFilter catalog={catalog} styles={styles} />
      {catalog.filteredIntegrationCatalog.length === 0 ? (
        <CatalogEmpty catalog={catalog} styles={styles} />
      ) : (
        <CatalogList appearance={appearance} catalog={catalog} styles={styles} />
      )}
      {appearance === "factories" ? (
        <FactoriesConnectModal catalog={catalog} />
      ) : (
        <LegacyConnectModal catalog={catalog} />
      )}
    </div>
  );
}

function CatalogLoading({ appearance }: { appearance: CatalogAppearance }) {
  if (appearance === "factories") {
    return <p className="text-[13px] text-muted-foreground">Loading integrations...</p>;
  }
  return (
    <div className="pt-6">
      <div className="flex h-32 items-center justify-center">
        <p className="text-gray-500 dark:text-gray-400">Loading integrations...</p>
      </div>
    </div>
  );
}

function CatalogFilter({ catalog, styles }: { catalog: CatalogState; styles: CatalogStyles }) {
  return (
    <div className="relative mb-4">
      <Search className={styles.searchIcon} />
      <Input
        type="text"
        value={catalog.filterQuery}
        onChange={(event) => catalog.setFilterQuery(event.target.value)}
        placeholder="Filter integrations..."
        className="pr-9 pl-9"
      />
      {catalog.filterQuery.length > 0 ? (
        <button
          type="button"
          onClick={() => catalog.setFilterQuery("")}
          className={styles.clearButton}
          aria-label="Clear filter"
        >
          <X className={styles.clearIcon} />
        </button>
      ) : null}
    </div>
  );
}

function CatalogEmpty({ catalog, styles }: { catalog: CatalogState; styles: CatalogStyles }) {
  return (
    <div className="py-12 text-center">
      <Plug className={styles.emptyIcon} />
      <p className={styles.emptyTitle}>
        {catalog.integrationCatalog.length === 0 ? "No integrations available." : "No integrations match your filter."}
      </p>
      {catalog.isIntegrationSurveyActive ? (
        <RequestIntegrationLink catalog={catalog} className={styles.requestWrap} styles={styles} />
      ) : null}
    </div>
  );
}

function CatalogList({
  appearance,
  catalog,
  styles,
}: {
  appearance: CatalogAppearance;
  catalog: CatalogState;
  styles: CatalogStyles;
}) {
  return (
    <div className={styles.list}>
      {catalog.filteredIntegrationCatalog.map((item) => (
        <CatalogProviderCard
          key={item.providerName}
          appearance={appearance}
          catalog={catalog}
          item={item}
          styles={styles}
        />
      ))}
      {catalog.isIntegrationSurveyActive ? (
        <RequestIntegrationLink catalog={catalog} className={styles.footerRequest} styles={styles} />
      ) : null}
    </div>
  );
}

function CatalogProviderCard({
  appearance,
  catalog,
  item,
  styles,
}: {
  appearance: CatalogAppearance;
  catalog: CatalogState;
  item: IntegrationCatalogItem;
  styles: CatalogStyles;
}) {
  return (
    <section
      id={`integration-${item.providerName}`}
      className={cn(styles.card, "scroll-mt-8")}
      data-testid={`integration-card-${item.providerName}`}
    >
      <div className={styles.cardHeader}>
        <div className="flex items-start gap-3">
          <div className="mt-0.5 flex size-8 items-center justify-center">
            <IntegrationIcon
              integrationName={item.providerName}
              iconSlug={item.integrationDef?.icon}
              className={styles.icon}
            />
          </div>
          <div>
            <h3 className={styles.title}>{item.providerLabel}</h3>
            {item.integrationDef?.description ? (
              <p className={styles.description}>{item.integrationDef.description}</p>
            ) : null}
          </div>
        </div>
        {usesHostedGitHubAppInstall(item.integrationDef ?? undefined) ? (
          <GitHubConnectControls
            organizationId={catalog.organizationId}
            definition={item.integrationDef ?? undefined}
            canCreateIntegrations={catalog.canCreateIntegrations}
            permissionsLoading={catalog.permissionsLoading}
            onConnect={() => item.integrationDef && catalog.handleConnectClick(item.integrationDef)}
            onCreatePrivateApp={() => catalog.handlePrivateAppClick(item.integrationDef ?? undefined)}
            allowPrivateApp={appearance !== "factories"}
          />
        ) : (
          <PermissionTooltip
            allowed={Boolean(item.integrationDef) && (catalog.canCreateIntegrations || catalog.permissionsLoading)}
            message={
              item.integrationDef
                ? "You don't have permission to connect integrations."
                : "This integration provider is no longer available for new connections."
            }
          >
            <Button
              variant="default"
              size="sm"
              onClick={() => item.integrationDef && catalog.handleConnectClick(item.integrationDef)}
              className="self-start"
              disabled={!item.integrationDef || !catalog.canCreateIntegrations}
            >
              {item.integrationDef ? "Connect" : "Unavailable"}
            </Button>
          </PermissionTooltip>
        )}
      </div>
      {item.instances.length > 0 ? (
        <div className={styles.instancesWrap}>
          <p className={styles.instanceCount}>
            {item.instances.length} connected instance{item.instances.length === 1 ? "" : "s"}
          </p>
          {item.instances.map((integration) => (
            <CatalogInstanceRow
              key={integration.metadata?.id}
              appearance={appearance}
              catalog={catalog}
              integration={integration}
              styles={styles}
            />
          ))}
        </div>
      ) : null}
    </section>
  );
}

function CatalogInstanceRow({
  appearance,
  catalog,
  integration,
  styles,
}: {
  appearance: CatalogAppearance;
  catalog: CatalogState;
  integration: IntegrationCatalogItem["instances"][number];
  styles: CatalogStyles;
}) {
  const state = integration.status?.state;
  return (
    <div className={styles.instanceRow}>
      <Plug className={`size-4 shrink-0 ${instancePlugClass(state, styles)}`} />
      <span className={instanceStatusLabelClass(appearance, state, styles)}>{integrationStatusLabel(state)}</span>
      <p className={styles.instanceName}>{integration.metadata?.name}</p>
      <div className="ml-auto">
        <PermissionTooltip
          allowed={catalog.canUpdateIntegrations || catalog.permissionsLoading}
          message="You don't have permission to update integrations."
        >
          <Button
            variant="outline"
            size="sm"
            onClick={() =>
              catalog.openInstance(
                integration.metadata?.integrationName,
                integration.metadata?.id,
                Boolean(integration.status?.setupState?.currentStep),
              )
            }
            disabled={!catalog.canUpdateIntegrations}
          >
            Configure
          </Button>
        </PermissionTooltip>
      </div>
    </div>
  );
}

function RequestIntegrationLink({
  catalog,
  className,
  styles,
}: {
  catalog: CatalogState;
  className: string;
  styles: CatalogStyles;
}) {
  return (
    <p className={className}>
      {styles.requestPrompt}
      <button type="button" onClick={catalog.handleRequestIntegration} className={styles.requestButton}>
        Request it
      </button>
    </p>
  );
}

function ConnectFields({ catalog }: { catalog: CatalogState }) {
  const selected = catalog.selectedIntegration;
  if (!selected) {
    return null;
  }
  return (
    <>
      {catalog.selectedInstructions ? <IntegrationInstructions description={catalog.selectedInstructions} /> : null}
      <div>
        <Label className="mb-2">
          Integration name
          <span className="ml-1">*</span>
        </Label>
        <Input
          type="text"
          value={catalog.integrationName}
          onChange={(event) => catalog.setIntegrationName(event.target.value)}
          placeholder="e.g., my-app-integration"
          required
          disabled={!catalog.canCreateIntegrations}
        />
        <p className="mt-2 text-[12px] text-muted-foreground">A unique name for this integration.</p>
      </div>
      {selected.configuration && selected.configuration.length > 0 ? (
        <div className="space-y-4">
          {selected.configuration
            .filter((field) => Boolean(field.name))
            .map((field) => (
              <ConfigurationFieldRenderer
                key={field.name!}
                field={field}
                value={catalog.configuration[field.name!]}
                onChange={(value) => catalog.setConfiguration({ ...catalog.configuration, [field.name!]: value })}
                allValues={catalog.configuration}
                organizationId={catalog.organizationId}
              />
            ))}
        </div>
      ) : null}
    </>
  );
}

function ConnectErrors({ catalog }: { catalog: CatalogState }) {
  if (!catalog.createIntegrationMutation.isError) {
    return null;
  }
  if (catalog.createIntegrationNotice) {
    return <UsageLimitAlert notice={catalog.createIntegrationNotice} className="mt-4" />;
  }
  return (
    <Alert variant="destructive" className="mt-4">
      <AlertTitle>Unable to create integration</AlertTitle>
      <AlertDescription>
        Failed to create integration: {getApiErrorMessage(catalog.createIntegrationMutation.error)}
      </AlertDescription>
    </Alert>
  );
}

function FactoriesConnectModal({ catalog }: { catalog: CatalogState }) {
  return (
    <Dialog
      open={catalog.isModalOpen && Boolean(catalog.selectedIntegration)}
      onOpenChange={(open) => !open && catalog.handleCloseModal()}
    >
      <DialogContent className="max-h-[80vh] max-w-2xl overflow-y-auto">
        {catalog.selectedIntegration ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-3">
                <IntegrationIcon
                  integrationName={catalog.selectedIntegration.name}
                  iconSlug={catalog.selectedIntegration.icon}
                  className="size-6 text-muted-foreground"
                />
                Connect {catalog.selectedIntegration.label || catalog.selectedIntegration.name}
              </DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <ConnectFields catalog={catalog} />
            </div>
            <div className="mt-6 flex justify-start gap-3">
              <Button
                onClick={() => void catalog.handleConnect()}
                disabled={
                  catalog.createIntegrationMutation.isPending ||
                  !catalog.integrationName.trim() ||
                  !catalog.canCreateIntegrations
                }
                className="flex items-center gap-2"
              >
                {catalog.createIntegrationMutation.isPending ? (
                  <>
                    <Loader2 className="size-4 animate-spin" />
                    Connecting...
                  </>
                ) : (
                  "Connect"
                )}
              </Button>
              <Button
                variant="outline"
                onClick={catalog.handleCloseModal}
                disabled={catalog.createIntegrationMutation.isPending}
              >
                Cancel
              </Button>
            </div>
            <ConnectErrors catalog={catalog} />
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function LegacyConnectModal({ catalog }: { catalog: CatalogState }) {
  if (!catalog.isModalOpen || !catalog.selectedIntegration) {
    return null;
  }
  const selected = catalog.selectedIntegration;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className={cn(settingsModalClassName, "max-h-[80vh] max-w-2xl overflow-y-auto")}>
        <div className="p-6">
          <div className="mb-6 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <IntegrationIcon
                integrationName={selected.name}
                iconSlug={selected.icon}
                className="h-6 w-6 text-gray-500 dark:text-gray-400"
              />
              <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100">
                Connect {selected.label || selected.name}
              </h3>
            </div>
            <button
              onClick={catalog.handleCloseModal}
              className="text-gray-500 hover:text-gray-800 dark:hover:text-gray-300"
              disabled={catalog.createIntegrationMutation.isPending}
            >
              <Icon name="x" size="sm" />
            </button>
          </div>
          <div className="space-y-4">
            <ConnectFields catalog={catalog} />
          </div>
          <div className="mt-6 flex justify-start gap-3">
            <Button
              color="blue"
              onClick={() => void catalog.handleConnect()}
              disabled={
                catalog.createIntegrationMutation.isPending ||
                !catalog.integrationName.trim() ||
                !catalog.canCreateIntegrations
              }
              className="flex items-center gap-2"
            >
              {catalog.createIntegrationMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Connecting...
                </>
              ) : (
                "Connect"
              )}
            </Button>
            <Button
              variant="outline"
              onClick={catalog.handleCloseModal}
              disabled={catalog.createIntegrationMutation.isPending}
            >
              Cancel
            </Button>
          </div>
          <ConnectErrors catalog={catalog} />
        </div>
      </div>
    </div>
  );
}
