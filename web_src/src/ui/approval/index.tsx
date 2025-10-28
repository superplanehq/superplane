import { CircleDashedIcon } from "lucide-react";
import React from "react";
import { ApprovalItem, type ApprovalItemProps } from "../approvalItem";
import { CollapsedComponent } from "../collapsedComponent";
import { ComponentHeader } from "../componentHeader";
import { ItemGroup } from "../item";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";

export interface AwaitingEvent {
  title: string;
  subtitle?: string;
}

export interface ApprovalProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconBackground?: string;
  iconColor?: string;
  headerColor: string;
  title: string;
  description?: string;
  approvals: ApprovalItemProps[];
  awaitingEvent?: AwaitingEvent;
  collapsedBackground?: string;
  receivedAt?: Date;
  zeroStateText?: string;
  collapsed?: boolean;
  selected?: boolean;
}

export const Approval: React.FC<ApprovalProps> = ({
  iconSrc,
  iconSlug,
  iconBackground,
  iconColor,
  headerColor,
  title,
  description,
  collapsed = false,
  collapsedBackground,
  receivedAt,
  approvals,
  awaitingEvent,
  zeroStateText = "No events yet",
  selected = false,
  onRun,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  const calcRelativeTimeFromDiff = (diff: number) => {
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    if (days > 0) {
      return `${days}d`;
    } else if (hours > 0) {
      return `${hours}h`;
    } else if (minutes > 0) {
      return `${minutes}m`;
    } else {
      return `${seconds}s`;
    }
  };

  const timeAgo = React.useMemo(() => {
    if (!receivedAt) return null;
    const now = new Date();
    const diff = now.getTime() - receivedAt.getTime();
    return calcRelativeTimeFromDiff(diff);
  }, [receivedAt]);

  if (collapsed) {
    return (
      <SelectionWrapper selected={selected}>
        <CollapsedComponent
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconColor={iconColor}
          iconBackground={iconBackground}
          title={title}
          collapsedBackground={collapsedBackground}
          shape="rounded"
          onRun={onRun}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        />
      </SelectionWrapper>
    );
  }

  return (
    <SelectionWrapper selected={selected}>
      <div className="flex flex-col border-2 border-border rounded-md w-[30rem] bg-white">
        <ComponentHeader
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        iconBackground={iconBackground}
        iconColor={iconColor}
        headerColor={headerColor}
        title={title}
        description={description}
        onRun={onRun}
        onDuplicate={onDuplicate}
        onDeactivate={onDeactivate}
        onToggleView={onToggleView}
        onDelete={onDelete}
        isCompactView={isCompactView}
      />

      <div className="px-4 py-3">
        {awaitingEvent ? (
          <>
            <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
              <span className="uppercase text-sm font-medium">
                Awaiting Approval
              </span>
              <span className="text-sm">{timeAgo}</span>
            </div>

            <div
              className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md bg-orange-200 mb-4`}
            >
              <div className="flex items-center gap-2 w-[80%] text-amber-800">
                <div
                  className={`w-5 h-5 rounded-full flex items-center justify-center`}
                >
                  <CircleDashedIcon size={20} className="text-amber-800" />
                </div>
                <span className="truncate text-sm">{awaitingEvent.title}</span>
              </div>
              {awaitingEvent.subtitle && (
                <span className="text-sm no-wrap whitespace-nowrap w-[20%] text-amber-800">
                  {awaitingEvent.subtitle}
                </span>
              )}
            </div>

            <ItemGroup className="w-full">
              {approvals.map((approval, index) => (
                <React.Fragment key={`${approval.title}-${index}`}>
                  <ApprovalItem {...approval} />
                </React.Fragment>
              ))}
            </ItemGroup>
          </>
        ) : (
          <div className="flex items-center justify-center px-2 py-4 rounded-md bg-gray-50 border border-dashed border-gray-300">
            <span className="text-sm text-gray-400">{zeroStateText}</span>
          </div>
        )}
      </div>
      </div>
    </SelectionWrapper>
  );
};
