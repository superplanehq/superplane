import { EventCard } from './EventCard';
import { SuperplaneStageEvent } from "@/api-client";

interface EventSectionProps {
  title: string;
  icon: string;
  iconColor: string;
  events: SuperplaneStageEvent[];
  variant: 'pending' | 'waiting' | 'processed';
  maxVisible?: number;
  emptyMessage: string;
  emptyIcon: string;
  onApprove?: (eventId: string, stageId: string) => void;
  stageId?: string;
  executionRunning?: boolean;
}

export const EventSection = ({
  title,
  icon,
  iconColor,
  events,
  variant,
  maxVisible = 3,
  emptyMessage,
  emptyIcon,
  onApprove,
  stageId,
  executionRunning
}: EventSectionProps) => {
  if (events.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200">
        <div className="p-4 border-b border-gray-200">
          <h4 className="text-sm font-medium text-gray-700 flex items-center">
            <span className={`material-symbols-outlined ${iconColor} mr-2`}>{icon}</span>
            {title} (0)
          </h4>
        </div>
        <div className="p-4">
          <div className="text-center py-6 text-gray-500">
            <div className="material-symbols-outlined text-4xl mb-2 opacity-50">{emptyIcon}</div>
            <div className="text-sm">{emptyMessage}</div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200">
      <div className="p-4 border-b border-gray-200">
        <h4 className="text-sm font-medium text-gray-700 flex items-center">
          <span className={`material-symbols-outlined ${iconColor} mr-2`}>{icon}</span>
          {title} ({events.length})
        </h4>
      </div>
      <div className="p-4">
        <div className="space-y-3">
          {events.slice(0, maxVisible).map((event) => (
            <EventCard
              key={event.id}
              eventId={event.id!}
              createdAt={event.createdAt!}
              state={event.state!}
              stateReason={event.stateReason}
              approvals={event.approvals}
              variant={variant}
              onApprove={onApprove && stageId ? () => onApprove(event.id!, stageId) : undefined}
              executionRunning={executionRunning}
            />
          ))}
          {events.length > maxVisible && (
            <div className="text-center">
              <button className={`text-sm ${iconColor} hover:opacity-80`}>
                View {events.length - maxVisible} more {variant}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};