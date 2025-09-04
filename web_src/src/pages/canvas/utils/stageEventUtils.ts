import { ExecutionWithEvent } from '../store/types';

export const formatRelativeTime = (dateString: string | undefined, abbreviated?: boolean) => {
  if (!dateString) return 'N/A'

  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  
  if (abbreviated) {
    if (Math.abs(diffSeconds) < 60) {
      return `${Math.abs(diffSeconds)}s ago`;
    } else if (Math.abs(diffMinutes) < 60) {
      return `${Math.abs(diffMinutes)}m ago`;
    } else if (Math.abs(diffHours) < 24) {
      return `${Math.abs(diffHours)}h ago`;
    } else {
      return `${Math.abs(diffDays)}d ago`;
    }
  } else {
    const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
    
    if (Math.abs(diffSeconds) < 60) {
      return rtf.format(-diffSeconds, 'second');
    } else if (Math.abs(diffMinutes) < 60) {
      return rtf.format(-diffMinutes, 'minute');
    } else if (Math.abs(diffHours) < 24) {
      return rtf.format(-diffHours, 'hour');
    } else {
      return rtf.format(-diffDays, 'day');
    }
  }
};

export const formatFullTimestamp = (dateString: string | undefined) => {
  if (!dateString) return 'N/A';
  const date = new Date(dateString);
  const timeFormat = date.toLocaleTimeString('en-US', { 
    hour: '2-digit', 
    minute: '2-digit', 
    hour12: false 
  });
  const dateFormat = date.toLocaleDateString('en-GB', { 
    day: '2-digit', 
    month: 'short', 
    year: '2-digit' 
  });
  return `${timeFormat} - ${dateFormat}`;
};
  
export const getExecutionStatusIcon = (state: string, result?: string) => {
  switch (state) {
    case 'STATE_PENDING': return '⏳';
    case 'STATE_STARTED': return '🔄';
    case 'STATE_FINISHED':
      return result === 'RESULT_PASSED' ? '✅' : result === 'RESULT_FAILED' ? '❌' : '⚪';
    default: return '⚪';
  }
};

export const getExecutionStatusColor = (state: string, result?: string) => {
  switch (state) {
    case 'STATE_PENDING': return 'text-amber-600 bg-amber-50';
    case 'STATE_STARTED': return 'text-blue-600 bg-blue-50';
    case 'STATE_FINISHED':
      return result === 'RESULT_PASSED' ? 'text-green-600 bg-green-50' : 
              result === 'RESULT_FAILED' ? 'text-red-600 bg-red-50' : 'text-gray-600 bg-gray-50';
    default: return 'text-gray-600 bg-gray-50';
  }
};

export const formatExecutionDuration = (startDate: string | undefined, endDate: string | undefined) => {
  if (!startDate || !endDate) return 'N/A';

  const start = new Date(startDate);
  const end = new Date(endDate);
  const diffMs = end.getTime() - start.getTime();

  if (diffMs < 0) return 'N/A';

  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffDays > 0) {
    return `${diffDays}d`;
  } else if (diffHours > 0) {
    return `${diffHours}h`;
  } else if (diffMinutes > 0) {
    return `${diffMinutes}m`;
  } else {
    return `${diffSeconds}s`;
  }
};

export const formatDuration = (startedAt?: string, finishedAt?: string) => {
  if (!startedAt || !finishedAt) {
    return "-";
  }
  const duration = new Date(finishedAt).getTime() - new Date(startedAt).getTime();
  const hours = Math.floor(duration / (1000 * 60 * 60));
  const prefixHours = hours >= 10 ? `${hours}h ` : `0${hours}h`;
  const minutes = Math.floor((duration % (1000 * 60 * 60)) / (1000 * 60));
  const prefixMinutes = minutes >= 10 ? `${minutes}m ` : `0${minutes}m`;
  const seconds = Math.floor((duration % (1000 * 60)) / 1000);
  const prefixSeconds = seconds >= 10 ? `${seconds}s` : `0${seconds}s`;
  return `${prefixHours} ${prefixMinutes} ${prefixSeconds}`;
};

interface UserData {
  metadata?: {
    id?: string;
    email?: string;
  };
  spec?: {
    displayName?: string;
  };
}

interface Approval {
  approvedAt?: string;
  approvedBy?: string;
}

export const getMinApprovedAt = (execution: ExecutionWithEvent) => {
  if (!execution.event.approvals?.length)
    return undefined;

  return execution.event.approvals.reduce((min: string, approval: Approval) => {
    if (approval.approvedAt && new Date(approval.approvedAt).getTime() < new Date(min).getTime()) {
      return approval.approvedAt;
    }
    return min;
  }, execution.event.approvals[0].approvedAt!);
};

export const getApprovalsNames = (execution: ExecutionWithEvent, userDisplayNames: Record<string, string>) => {
  const names: string[] = [];
  execution.event.approvals?.forEach((approval: Approval) => {
    if (approval.approvedBy) {
      names.push(userDisplayNames[approval.approvedBy]);
    }
  });
  return names.join(', ');
};

export const getCancelledByName = (execution: ExecutionWithEvent, userDisplayNames: Record<string, string>) => {
  if (!execution.event.cancelledBy) {
    return undefined;
  }
  return userDisplayNames[execution.event.cancelledBy] || execution.event.cancelledBy;
};

export const mapExecutionOutputs = (execution: ExecutionWithEvent) => {
  const map: Record<string, string> = {};
  execution.outputs?.forEach((output) => {
    if (!output.name) {
      return;
    }

    map[output.name!] = output.value!;
  });

  return map;
};

export const mapExecutionEventInputs = (execution: ExecutionWithEvent) => {
  const map: Record<string, string> = {};
  execution.event.inputs?.forEach((input) => {
    if (!input.name) {
      return;
    }

    map[input.name!] = input.value!;
  });

  return map;
};

export const createUserDisplayNames = (orgUsers: UserData[]) => {
  const map: Record<string, string> = {};
  orgUsers.forEach(user => {
    if (user.metadata?.id) {
      map[user.metadata.id] = user.spec?.displayName || user.metadata?.email || user.metadata.id;
    }
  });
  return map;
};