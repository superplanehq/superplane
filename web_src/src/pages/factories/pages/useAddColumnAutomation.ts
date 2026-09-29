import { organizationsDescribeIntegration } from "@/api-client";
import { useCreateFactoryAutomation } from "@/hooks/useFactoryData";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_CUSTOM_AUTOMATIONS, FEATURE_FACTORY_RISK_SCORE } from "@/lib/experimentalFeatures";
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

export function useAddColumnAutomation(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lineId?: string;
  appRepository: string;
  backlogRepository: string;
  defaultBranch: string;
  githubIntegrationId: string;
  automationsFor: (key: ColumnKey) => ColumnAutomation[];
}) {
  const [column, setColumn] = useState<ColumnKey | null>(null);
  const [naming, setNaming] = useState(false);
  const navigate = useNavigate();
  const createAutomation = useCreateFactoryAutomation(args.organizationId, args.factoryId);
  const { installFactory, isInstalling } = useInstallFactory({ organizationId: args.organizationId });
  const experimentalFeatures = useExperimentalFeature(args.organizationId);
  const allowCustom = experimentalFeatures.has(FEATURE_FACTORY_CUSTOM_AUTOMATIONS);
  const allowRiskScore = experimentalFeatures.has(FEATURE_FACTORY_RISK_SCORE);
  const catalogOptions = { allowCustom, allowRiskScore };

  const catalog = column ? catalogForColumn(column, catalogOptions) : [];
  const takenIds = column ? takenCatalogIds(args.automationsFor(column), catalog) : [];

  const closePicker = () => {
    setColumn(null);
    setNaming(false);
  };

  const openPicker = (key: ColumnKey) => {
    const nextCatalog = catalogForColumn(key, catalogOptions);
    const nextTaken = takenCatalogIds(args.automationsFor(key), nextCatalog);
    setColumn(key);
    setNaming(onlyCustomCatalogRemains(nextCatalog, nextTaken));
  };

  const onSelect = async (entry: ColumnAutomationCatalogEntry) => {
    if (!column) {
      return;
    }
    if (entry.kind === "pr-discussion" || entry.kind === "pr-checks") {
      openPRFeedbackSetup(entry.id, args, navigate, closePicker);
      return;
    }
    if (entry.kind === "pr-closure") {
      await installBundledCanvas(args, installFactory, navigate, closePicker, {
        factoryId: "pr-closure",
        missingGitHubMessage: "Connect GitHub in workspace setup before you add pull request closure.",
      });
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

function openPRFeedbackSetup(
  catalogId: string,
  args: { organizationId: string; factoryKey: string; lineId?: string },
  navigate: (path: string) => void,
  close: () => void,
) {
  const sourceId = catalogId === "checks" ? "checks" : "discussion";
  const href = args.lineId
    ? factoryPRFeedbackSetupPath(
        args.organizationId,
        args.factoryKey,
        args.lineId,
        prFeedbackSetupKindFromSourceId(sourceId),
      )
    : undefined;
  close();
  if (href) {
    navigate(href);
  }
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
  },
  installFactory: ReturnType<typeof useInstallFactory>["installFactory"],
  navigate: (path: string) => void,
  close: () => void,
  options: { factoryId: string; missingGitHubMessage: string },
) {
  if (!args.githubIntegrationId) {
    showErrorToast(options.missingGitHubMessage);
    return;
  }
  const githubInstallationName = await installationName(args.organizationId, args.githubIntegrationId);
  if (!githubInstallationName) {
    showErrorToast(options.missingGitHubMessage);
    return;
  }
  try {
    const installed = await installFactory({
      factoryId: options.factoryId,
      workspaceFactoryId: args.factoryId,
      integrations: { github: { id: args.githubIntegrationId, name: githubInstallationName, ready: true } },
      installParams: {
        appRepository: args.appRepository,
        backlogRepository: args.backlogRepository,
        defaultBranch: args.defaultBranch,
      },
      startingTaskPrompt: "",
      navigateOnComplete: false,
      startInitialRun: false,
    });
    close();
    if (!installed?.canvasId) {
      return;
    }
    navigate(
      factoryAppConfigurePath(args.organizationId, args.factoryKey, installed.canvasId, {
        from: "lines",
        lineId: args.lineId,
      }),
    );
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
