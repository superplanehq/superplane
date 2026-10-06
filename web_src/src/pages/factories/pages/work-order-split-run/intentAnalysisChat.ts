import type { ReactNode } from "react";

import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";

import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import type { CreateWithAgentView } from "../createWithAgentTypes";
import type { ComposerScore } from "./PlanningReview";
import type { PlanChipStatus } from "./planChipStatus";
import type { ComposerCreditVerdict } from "./splitRunFooter";

export type IntentAnalysisChat = {
  planningReviewEnabled?: boolean;
  organizationId: string;
  factoryId?: string;
  view: CreateWithAgentView;
  composer: string;
  composerError?: string;
  canSend: boolean;
  isUploading?: boolean;
  onComposerChange: (value: string) => void;
  onSend: (text?: string) => void | Promise<boolean>;
  onUploadFiles?: (files: FileList | File[]) => Promise<UploadedWorkOrderFile[]>;
  onSubmitSurvey: (text: string) => void;
  planTitle?: string;
  planPaneOpen?: boolean;
  onTogglePlan?: () => void;
  canTogglePlan?: boolean;
  clarity?: ComposerScore;
  confidence?: ComposerScore;
  reviewMetrics?: WorkOrderCheckPresentation[];
  showClarity?: boolean;
  showConfidence?: boolean;
  planStatus?: PlanChipStatus;
  isAnalyzing?: boolean;
  creditVerdict?: ComposerCreditVerdict;
  prioritizeImplementation?: boolean;
  startDiscouraged?: boolean;
  closedDecision?: ReactNode;
  modelSelect?: ReactNode;
};
