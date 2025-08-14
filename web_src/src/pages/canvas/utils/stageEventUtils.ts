export const formatRelativeTime = (dateString: string | undefined) => {
  if (!dateString) return 'N/A'

  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  
  const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
  
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  
  if (Math.abs(diffSeconds) < 60) {
    return rtf.format(-diffSeconds, 'second');
  } else if (Math.abs(diffMinutes) < 60) {
    return rtf.format(-diffMinutes, 'minute');
  } else if (Math.abs(diffHours) < 24) {
    return rtf.format(-diffHours, 'hour');
  } else {
    return rtf.format(-diffDays, 'day');
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