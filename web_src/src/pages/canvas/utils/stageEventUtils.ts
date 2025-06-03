export const formatRelativeTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  
    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${diffDays}d ago`;
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