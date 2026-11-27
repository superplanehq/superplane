import { organizationsDescribeIntegration } from "@/api-client";
import { useQueryClient } from "@tanstack/react-query";
import { factoryVCSProvider } from "@/pages/home/factories";
import { factoryAppsKey, useCreateFactoryAutomation } from "@/hooks/useFactoryData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_REVIEW_REQUEST } from "@/lib/experimentalFeatures";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useInstallFactory } from "@/pages/home/useInstallFactory";
import { useState } from "react";
import { useNavigate } from "react-router";

import {
  catalogForColumn,
  onlyCustomCatalogRemains,
  pickerCatalogForColumn,
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
import type { CustomAutomationColumn } from "../lib/customAutomation";

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
  const [columnTitle, setColumnTitle] = useState("");
  const [naming, setNaming] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const createAutomation = useCreateFactoryAutomation(args.organizationId, args.factoryId);
  const { installFactory, isInstalling } = useInstallFactory({ organizationId: args.organizationId });
  const experimentalFeatures = useExperimentalFeature(args.organizationId);
  const allowReviewRequest = experimentalFeatures.has(FEATURE_FACTORY_REVIEW_REQUEST);
  const catalogOptions = { allowReviewRequest, vcsProvider: args.vcsProvider };

  const pickerCatalog = column
    ? pickerCatalogForColumn(column, catalogOptions)
    : { catalog: [], unavailableReasons: {} };
  const catalog = pickerCatalog.catalog;
  const takenIds = column ? takenCatalogIds(installedPickerAutomations(column, args.automationsFor), catalog) : [];

  const closePicker = () => {
    setColumn(null);
    setColumnTitle("");
    setNaming(false);
  };

  const openPicker = (key: ColumnKey, title = "") => {
    const nextCatalog = catalogForColumn(key, catalogOptions);
    const nextTaken = takenCatalogIds(args.automationsFor(key), nextCatalog);
    const setupColumn = customAutomationSetupColumn(key);
    const customStage = key.startsWith("custom-");
    if (setupColumn !== null && onlyCustomCatalogRemains(nextCatalog, nextTaken) && args.lineId) {
      navigate(factoryCustomAutomationSetupPath(args.organizationId, args.factoryKey, undefined, setupColumn));
      return;
    }
    setColumn(key);
    setColumnTitle(title);
    setNaming(!customStage && onlyCustomCatalogRemains(nextCatalog, nextTaken));
  };

  const onSelect = async (entry: ColumnAutomationCatalogEntry) => {
    if (!column || pickerCatalog.unavailableReasons[entry.id] || takenIds.includes(entry.id)) {
      return;
    }
    if (
      entry.kind === "pr-discussion" ||
      entry.kind === "pr-checks" ||
      entry.kind === "custom-verification" ||
      (entry.kind === "custom" && column.startsWith("custom-"))
    ) {
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
        navigate(factoryRiskScoreSetupPath(args.organizationId, args.factoryKey));
      }
      return;
    }
    if (entry.kind === "analysis" || entry.kind === "agent-step") {
      await restoreColumnAutomation(entry, column, args.lineId, createAutomation.mutateAsync, closePicker);
      return;
    }
    if (entry.kind === "custom") setNaming(true);
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
    pickerColumn: column,
    pickerTitle: columnTitle ? `Add an automation for ${columnTitle}` : undefined,
    pickerOpen: column !== null && !naming,
    naming,
    openPicker,
    closePicker,
    catalog: catalog.filter((entry) => column !== "backlog" || entry.kind !== "intake"),
    takenIds,
    unavailableReasons: pickerCatalog.unavailableReasons,
    onSelect,
    createNamed,
    isPending: createAutomation.isPending || isInstalling,
  };
}

function installedPickerAutomations(column: ColumnKey, automationsFor: (key: ColumnKey) => ColumnAutomation[]) {
  const keys: ColumnKey[] = ["backlog", "phase-0", "verify", "done", column];
  // Built-in templates are workspace-wide, but each stage has its own agent slot.
  return keys.flatMap((key) =>
    automationsFor(key).filter((automation) => automation.kind !== "agent-step" || key === column),
  );
}

async function restoreColumnAutomation(
  entry: ColumnAutomationCatalogEntry,
  column: ColumnKey,
  lineId: string | undefined,
  create: ReturnType<typeof useCreateFactoryAutomation>["mutateAsync"],
  close: () => void,
) {
  try {
    await create({
      name: entry.name,
      templateId: entry.kind === "analysis" ? "backlog" : "line-implementation",
      installParams: { lineId: lineId ?? "", stageKey: column },
    });
    close();
  } catch {
    showErrorToast("Could not add automation. Try again.");
  }
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

function customAutomationSetupColumn(key: ColumnKey): CustomAutomationColumn | undefined | null {
  if (key.startsWith("custom-")) return key as CustomAutomationColumn;
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
  if (entry.kind === "custom-verification" || entry.kind === "custom") {
    const setupColumn = customAutomationSetupColumn(column);
    return factoryCustomAutomationSetupPath(args.organizationId, args.factoryKey, undefined, setupColumn ?? undefined);
  }
  return factoryPRFeedbackSetupPath(args.organizationId, args.factoryKey, prFeedbackSetupKindFromSourceId(sourceId));
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
    columnKey:
      input.column === "verify" || input.column === "done" || input.column.startsWith("custom-")
        ? input.column
        : undefined,
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
