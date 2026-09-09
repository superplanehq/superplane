import type { FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";
import { CopyLinkButton } from "./CopyLinkButton";
import { WorkspacePageHeader } from "./layout/WorkspacePageHeader";
import { buildWorkOrderStatusActions } from "./lib/workOrderStatusActions";
import type { WorkOrderDisplayStatus } from "./lib/workOrderProgress";
import { WorkOrderOverflowMenu } from "./WorkOrderOverflowMenu";

interface WorkOrderDetailHeaderProps {
  orderTitle: string;
  /** Short identifier (e.g. `SP-42`). Rendered as a kicker above the title. */
  orderIdentifier?: string;
  /** Back link target (workspace board). Omit in the card dialog. */
  backHref?: string;
  /** Back link label. Defaults to Workspace. */
  backLabel?: string;
  displayStatus: WorkOrderDisplayStatus;
  isOpen: boolean;
  isDispatchable: boolean;
  isClosed: boolean;
  canClose: boolean;
  canManage: boolean;
  canCreate: boolean;
  isCompleting: boolean;
  isRejecting: boolean;
  isClosing: boolean;
  isUpdatingStatus: boolean;
  isDuplicating?: boolean;
  onClose: (result: FactoriesWorkOrderResult) => void;
  onStatusChange: (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => Promise<void>;
  onDuplicate: () => void;
  className?: string;
}

export function WorkOrderDetailHeader(props: WorkOrderDetailHeaderProps) {
  return (
    <WorkspacePageHeader
      className={props.className}
      variant="entity"
      backHref={props.backHref}
      backLabel={props.backHref ? (props.backLabel ?? "Workspace") : undefined}
      backTestId="work-order-detail-back"
      kicker={props.orderIdentifier}
      title={props.orderTitle}
      actions={
        <>
          <CopyLinkButton
            className="size-7 rounded-full text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:pointer-events-none disabled:opacity-50"
            iconClassName="size-3.5"
          />
          <HeaderOverflowMenu {...props} />
        </>
      }
    />
  );
}

function HeaderOverflowMenu(props: WorkOrderDetailHeaderProps) {
  const actions = buildWorkOrderStatusActions(props);
  const disabled = props.isClosing || props.isUpdatingStatus || props.isCompleting || props.isRejecting;

  return (
    <WorkOrderOverflowMenu
      canCreate={props.canCreate}
      isDuplicating={props.isDuplicating}
      onDuplicate={props.onDuplicate}
      actions={actions}
      onClose={props.onClose}
      onStatusChange={props.onStatusChange}
      disabled={disabled}
    />
  );
}
