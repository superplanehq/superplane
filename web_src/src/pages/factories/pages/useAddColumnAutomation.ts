import { organizationsDescribeIntegration } from "@/api-client";
import { useQueryClient } from "@tanstack/react-query";
import { factoryVCSProvider } from "@/pages/home/factories";
import { factoryAppsKey, useCreateFactoryAutomation } from "@/hooks/useFactoryData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_CUSTOM_AUTOMATIONS, FEATURE_FACTORY_REVIEW_REQUEST } from "@/lib/experimentalFeatures";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useInstallFactory } from "@/pages/home/useInstallFactory";
import { useState } from "react";
import { useNavigate } from "react-router";

import {
  catalogForColumn,
  onlyCustomCatalogRemains,
  takenCatalogIds,
  type ColumnAutomation,
  type ColumnAutomationCatalogEntry,
  type ColumnKey,
} from "../lib/columnAutomations";
import {
  factoryAppConfigurePath,
  factoryPRFeedbackSetupPath,
  factoryRiskScoreSetupPath,
  prFeedbackSetupKindFromSourceId,
} from "../lib/factoryPagePaths";
import { factoryCustomAutomationSetupPath } from "../lib/customAutomationPaths";

export function useAddColumnAutomation(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  appRepository: string;
  backlogRepository: string;
  defaultBranch: string;
  githubIntegrationId: string;
  vcsProvider?: string;
  automationsFor: (key: ColumnKey) => ColumnAutomation[];
}) {
  const [column, setColumn] = useState<ColumnKey | null>(null);
  const [naming, setNaming] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const createAutomation = useCreateFactoryAutomation(args.organizationId, args.factoryId);
  const { installFactory, isInstalling } = useInstallFactory({ organizationId: args.organizationId });
  const experimentalFeatures = useExperimentalFeature(args.organizationId);
  const allowCustom = experimentalFeatures.has(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
  const allowReviewRequest = experimentalFeatures.has(FEATURE_FACTORY_REVIEW_REQUEST);
  const catalogOptions = { allowCustom, allowReviewRequest, vcsProvider: args.vcsProvider };

  const catalog = column ? catalogForColumn(column, catalogOptions) : [];
  const takenIds = column ? takenCatalogIds(args.automationsFor(column), catalog) : [];

  const closePicker = () => {
    setColumn(null);
    setNaming(false);
  };

  const openPicker = (key: ColumnKey) => {
    const nextCatalog = catalogForColumn(key, catalogOptions);
    const nextTaken = takenCatalogIds(args.automationsFor(key), nextCatalog);
    const setupColumn = customAutomationSetupColumn(key);
    if (setupColumn !== null && onlyCustomCatalogRemains(nextCatalog, nextTaken) && args.lineId) {
      navigate(
        factoryCustomAutomationSetupPath(args.organizationId, args.factoryKey, args.lineId, undefined, setupColumn),
      );
      return;
    }
    setColumn(key);
    setNaming(onlyCustomCatalogRemains(nextCatalog, nextTaken));
  };

  const onSelect = async (entry: ColumnAutomationCatalogEntry) => {
    if (!column) {
      return;
    }
    if (entry.kind === "pr-discussion" || entry.kind === "pr-checks" || entry.kind === "custom-verification") {
      openFocusedVerifySetup(entry, column, args, navigate, closePicker);
      return;
    }
    if (entry.kind === "pr-closure") {
      const provider = factoryVCSProvider(args.vcsProvider);
      const host = provider === "bitbucket" ? "Bitbucket" : "GitHub";
      await installBundledCanvas(args, installFactory, queryClient, closePicker, {
        factoryId: "pr-closure",
        missingIntegrationMessage: `Connect ${host} in workspace setup before you add pull request closure.`,
      });
      return;
    }
    if (entry.kind === "visual-evidence" || entry.kind === "review-request") {
      const template = githubVerifyTemplate(entry.kind, args.vcsProvider, allowReviewRequest);
      if (template) await installBundledCanvas(args, installFactory, queryClient, closePicker, template);
      return;
    }
    if (entry.kind === "risk-score") {
      closePicker();
      if (args.lineId) {
        navigate(factoryRiskScoreSetupPath(args.organizationId, args.factoryKey, args.lineId));
      }
      return;
    }
    if (entry.kind !== "custom") {
      return;
    }
    setNaming(true);
  };

  const createNamed = async (name: string) => {
    if (!column) {
      return;
    }
    await createCustomAutomation({
      column,
      args,
      mutateAsync: createAutomation.mutateAsync,
      navigate,
      close: closePicker,
      name,
    });
  };

  return {
    allowCustom,
    pickerColumn: column,
    pickerOpen: column !== null && !naming,
    naming,
    openPicker,
    closePicker,
    catalog,
    takenIds,
    onSelect,
    createNamed,
    isPending: createAutomation.isPending || isInstalling,
  };
}

function githubVerifyTemplate(
  kind: "visual-evidence" | "review-request",
  vcsProvider: string | undefined,
  allowReviewRequest: boolean,
) {
  if (kind === "review-request" && !allowReviewRequest) return undefined;
  if (factoryVCSProvider(vcsProvider) !== "github") {
    if (kind === "visual-evidence") showErrorToast("Visual evidence is available for GitHub only.");
    return undefined;
  }
  const label = kind === "review-request" ? "Review request" : "visual evidence";
  return {
    factoryId: kind,
    provider: "github" as const,
    missingIntegrationMessage: `Connect GitHub in workspace setup before you add ${label}.`,
  };
}

function customAutomationSetupColumn(key: ColumnKey): "implement" | "verify" | "done" | undefined | null {
  if (key === "verify") return undefined;
  if (key === "done") return "done";
  if (key === "phase-0") return "implement";
  return null;
}

function openFocusedVerifySetup(
  entry: ColumnAutomationCatalogEntry,
  column: ColumnKey,
  args: { organizationId: string; factoryKey: string; lineId?: string },
  navigate: (path: string) => void,
  close: () => void,
) {
  const sourceId = entry.id === "checks" ? "checks" : "discussion";
  const href = args.lineId ? focusedVerifySetupPath(entry, column, args, sourceId) : undefined;
  close();
  if (href) {
    navigate(href);
  }
}

function focusedVerifySetupPath(
  entry: ColumnAutomationCatalogEntry,
  column: ColumnKey,
  args: { organizationId: string; factoryKey: string; lineId?: string },
  sourceId: "checks" | "discussion",
): string | undefined {
  if (!args.lineId) {
    return undefined;
  }
  if (entry.kind === "custom-verification") {
    const setupColumn = customAutomationSetupColumn(column);
    return factoryCustomAutomationSetupPath(
      args.organizationId,
      args.factoryKey,
      args.lineId,
      undefined,
      setupColumn ?? undefined,
    );
  }
  return factoryPRFeedbackSetupPath(
    args.organizationId,
    args.factoryKey,
    args.lineId,
    prFeedbackSetupKindFromSourceId(sourceId),
  );
}

async function installationName(organizationId: string, integrationId: string): Promise<string> {
  try {
    const response = await organizationsDescribeIntegration(
      withOrganizationHeader({
        organizationId,
        path: { id: organizationId, integrationId },
      }),
    );
    return response.data?.integration?.metadata?.name?.trim() ?? "";
  } catch {
    return "";
  }
}

async function installBundledCanvas(
  args: {
    organizationId: string;
    factoryId: string;
    factoryKey: string;
    lineId?: string;
    appRepository: string;
    backlogRepository: string;
    defaultBranch: string;
    githubIntegrationId: string;
    vcsProvider?: string;
  },
  installFactory: ReturnType<typeof useInstallFactory>["installFactory"],
  queryClient: ReturnType<typeof useQueryClient>,
  close: () => void,
  template: { factoryId: string; missingIntegrationMessage: string; provider?: "github" | "bitbucket" },
) {
  const provider = template.provider ?? factoryVCSProvider(args.vcsProvider);
  if (!args.githubIntegrationId) {
    showErrorToast(template.missingIntegrationMessage);
    return;
  }
  const vcsInstallationName = await installationName(args.organizationId, args.githubIntegrationId);
  if (!vcsInstallationName) {
    showErrorToast(template.missingIntegrationMessage);
    return;
  }
  try {
    const installed = await installFactory({
      factoryId: template.factoryId,
      vcsProvider: provider,
      workspaceFactoryId: args.factoryId,
      integrations: { [provider]: { id: args.githubIntegrationId, name: vcsInstallationName, ready: true } },
      installParams: {
        appRepository: args.appRepository,
        backlogRepository: args.backlogRepository,
        defaultBranch: args.defaultBranch,
      },
      startingTaskPrompt: "",
      navigateOnComplete: false,
      startInitialRun: false,
    });
    if (!installed?.canvasId) {
      return;
    }
    await queryClient.invalidateQueries({ queryKey: factoryAppsKey(args.organizationId, args.factoryId) });
    close();
  } catch {
    // useInstallFactory already reports the error.
  }
}

async function createCustomAutomation(input: {
  column: ColumnKey;
  args: { organizationId: string; factoryKey: string; lineId?: string };
  mutateAsync: (body: { name?: string; columnKey?: string }) => Promise<{ id?: string }>;
  navigate: (path: string) => void;
  close: () => void;
  name: string;
}) {
  const automation = await input.mutateAsync({
    name: input.name,
    columnKey: input.column === "verify" || input.column === "done" ? input.column : undefined,
  });
  input.close();
  if (!automation.id) {
    return;
  }
  input.navigate(
    factoryAppConfigurePath(input.args.organizationId, input.args.factoryKey, automation.id, {
      from: "lines",
      lineId: input.args.lineId,
    }),
  );
}
