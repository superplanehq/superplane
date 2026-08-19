import { createContext, useContext, type ComponentType } from "react";

import type { FactoriesFactoryLine } from "@/api-client";

/**
 * Storybook can replace the New work order footer actions (Owner, Save as
 * draft, Send to line). The live app leaves this empty and keeps today's
 * split header/footer layout.
 */
export interface CreateWorkOrderActionSlotProps {
  organizationId: string;
  assigneeIds: string[];
  lines: FactoriesFactoryLine[];
  isSaving: boolean;
  canDispatch: boolean;
  canSaveDraft: boolean;
  isSavingDraft: boolean;
  isSendingToLine: boolean;
  onAssigneeChange: (assigneeIds: string[]) => void;
  onSaveDraft: () => void;
  onSendToLine: (lineName: string) => void;
}

export type CreateWorkOrderActionSlot = ComponentType<CreateWorkOrderActionSlotProps>;

export const CreateWorkOrderActionSlotContext = createContext<CreateWorkOrderActionSlot | null>(null);

export function useCreateWorkOrderActionSlot() {
  return useContext(CreateWorkOrderActionSlotContext);
}
