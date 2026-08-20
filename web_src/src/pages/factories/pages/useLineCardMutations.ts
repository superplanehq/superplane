import type { FactoriesFactoryLine } from "@/api-client";
import { useCreateFactoryLine } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback, useState } from "react";
import { useNavigate } from "react-router";

import { duplicateFactoryLine } from "./duplicateFactoryLine";
import { editFactoryLinePath, factoryLineDetailPath } from "../lib/factoryPagePaths";
import type { LineCardActions } from "./lineCardActions";

/**
 * Wires the lines-list card menu to navigation, toasts, and `useCreateFactoryLine`.
 * Duplicate is a single create call (see `duplicateFactoryLine`) — lines have
 * no canvas/YAML staging step the way automations do.
 */
export function useLineCardMutations(args: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  lines: FactoriesFactoryLine[];
  canUpdate: boolean;
}) {
  const { organizationId, factoryId, factoryKey, lines, canUpdate } = args;
  const navigate = useNavigate();
  const createLine = useCreateFactoryLine(organizationId, factoryId);
  const [duplicatingLineId, setDuplicatingLineId] = useState<string | null>(null);

  const handleEditLine = useCallback(
    (line: FactoriesFactoryLine) => {
      if (!line.id) {
        return;
      }
      navigate(editFactoryLinePath(organizationId, factoryKey, line.id));
    },
    [factoryKey, navigate, organizationId],
  );

  const handleDuplicateLine = useCallback(
    async (line: FactoriesFactoryLine) => {
      if (!line.id || duplicatingLineId) {
        return;
      }
      setDuplicatingLineId(line.id);
      try {
        const existingNames = lines.map((entry) => entry.name).filter((name): name is string => Boolean(name));
        const created = await duplicateFactoryLine({
          line,
          createLine: (input) => createLine.mutateAsync(input),
          existingNames,
        });
        showSuccessToast("Line duplicated.");
        if (created.id) {
          navigate(factoryLineDetailPath(organizationId, factoryKey, created.id));
        }
      } catch (error) {
        showErrorToast(getApiErrorMessage(error, "Failed to duplicate line"));
      } finally {
        setDuplicatingLineId(null);
      }
    },
    [createLine, duplicatingLineId, factoryKey, lines, navigate, organizationId],
  );

  const actionsForLine = useCallback(
    (line: FactoriesFactoryLine): LineCardActions => ({
      onEdit: () => handleEditLine(line),
      onDuplicate: () => handleDuplicateLine(line),
      canEdit: canUpdate,
      canDuplicate: canUpdate,
      isDuplicating: Boolean(line.id) && duplicatingLineId === line.id,
    }),
    [canUpdate, duplicatingLineId, handleDuplicateLine, handleEditLine],
  );

  return { actionsForLine };
}
