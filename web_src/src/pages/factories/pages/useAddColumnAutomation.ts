import { useCreateFactoryAutomation } from "@/hooks/useFactoryData";
import { showErrorToast } from "@/lib/toast";
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

  const catalog = column ? catalogForColumn(column) : [];
  const takenIds = column ? takenCatalogIds(args.automationsFor(column), catalog) : [];

  const closePicker = () => {
    setColumn(null);
    setNaming(false);
  };

  const openPicker = (key: ColumnKey) => {
    const nextCatalog = catalogForColumn(key);
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
      await installPRClosure(args, installFactory, navigate, closePicker);
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

async function installPRClosure(
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
) {
  if (!args.githubIntegrationId) {
    showErrorToast("Connect GitHub in workspace setup before you add pull request closure.");
    return;
  }
  try {
    const installed = await installFactory({
      factoryId: "pr-closure",
      workspaceFactoryId: args.factoryId,
      integrations: { github: { id: args.githubIntegrationId, name: "GitHub", ready: true } },
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
