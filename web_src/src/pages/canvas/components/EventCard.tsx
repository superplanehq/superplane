import { SuperplaneStageEventApproval } from '@/api-client';
import { formatRelativeTime } from '../utils/stageEventUtils';

interface EventCardProps {
  eventId: string;
  createdAt: string;
  state: string;
  stateReason?: string;
  approvals?: SuperplaneStageEventApproval[];
  onApprove?: () => void;
  variant?: 'pending' | 'waiting' | 'processed';
  executionRunning?: boolean;
}

export const EventCard = ({ 
  eventId, 
  createdAt, 
  state, 
  stateReason, 
  approvals,
  onApprove,
  variant = 'pending',
  executionRunning = false
}: EventCardProps) => {
  const getVariantStyles = () => {
    switch (variant) {
      case 'pending':
        return {
          containerClass: 'bg-amber-50 border-amber-200',
          iconClass: 'text-amber-600',
          badgeClass: 'bg-amber-100 text-amber-800',
          icon: 'pending'
        };
      case 'waiting':
        return {
          containerClass: 'bg-blue-50 border-blue-200',
          iconClass: 'text-blue-600',
          badgeClass: 'bg-blue-100 text-blue-800',
          icon: 'hourglass_empty'
        };
      case 'processed':
        return {
          containerClass: 'bg-green-50 border-green-200',
          iconClass: 'text-green-600',
          badgeClass: 'bg-green-100 text-green-800',
          icon: 'check_circle'
        };
    }
  };

  const styles = getVariantStyles();

  return (
    <div className={`p-3 rounded-lg ${styles.containerClass} border`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <div className={`material-symbols-outlined ${styles.iconClass}`}>
            {styles.icon}
          </div>
          <div>
            <div className="text-sm font-medium text-gray-900">
              {formatRelativeTime(createdAt)}
            </div>
            <div className="text-xs text-gray-500">ID: {eventId.substring(0, 8)}...</div>
            {stateReason && (
              <div className={`text-xs mt-1 ${styles.iconClass}`}>
                Reason: {stateReason.replace('STATE_REASON_', '')}
              </div>
            )}
            {approvals && approvals.length > 0 && (
              <div className={`text-xs mt-1 ${styles.iconClass}`}>
                Approvals: {approvals.length}
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center space-x-2">
          {onApprove && variant === 'waiting' && (
            <button 
              onClick={executionRunning ? undefined : onApprove}
              disabled={executionRunning}
              className="px-3 py-1.5 text-xs font-medium text-white rounded-md transition-colors disabled:bg-gray-400 disabled:text-gray-500 disabled:cursor-not-allowed"
              style={{ backgroundColor: '#2563eb' }} 
              onMouseEnter={(e) => e.currentTarget.style.backgroundColor = '#1d4ed8'} 
              onMouseLeave={(e) => e.currentTarget.style.backgroundColor = '#2563eb'}
            >
              Approve
            </button>
          )}
          <div className={`text-xs px-2 py-1 rounded ${styles.badgeClass}`}>
            {state.replace('STATE_', '')}
          </div>
        </div>
      </div>
    </div>
  );
};