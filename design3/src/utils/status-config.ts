export interface StatusConfig {
  bgColor: string;
  borderColor: string;
  textColor: string;
  icon: string;
  iconColor: string;
}

export const getStatusConfig = (status: string): StatusConfig => {
  switch (status?.toLowerCase()) {
    case 'success':
    case 'passed':
      return {
        bgColor: 'bg-green-50 dark:bg-green-900/50',
        borderColor: 'border-t border-t-green-400 dark:border-t-green-700',
        textColor: 'text-green-700 dark:text-green-400',
        icon: 'check_circle',
        iconColor: 'text-green-500 dark:text-green-400',
      };
    case 'error':
    case 'failed':
      return {
        bgColor: 'bg-red-50 dark:bg-red-900/50',
        borderColor: 'border-t border-t-red-400 dark:border-t-red-700',
        textColor: 'text-red-700 dark:text-red-400',
        icon: 'cancel',
        iconColor: 'text-red-500 dark:text-red-400',
      };
    case 'running':
      return {
        bgColor: 'bg-blue-50 dark:bg-blue-900/50',
        borderColor: 'border-t border-t-blue-400 dark:border-t-blue-700',
        textColor: 'text-blue-700 dark:text-blue-400',
        icon: 'sync',
        iconColor: 'text-blue-500 dark:text-blue-400 animate-spin',
      };
    case 'pending':
    case 'queued':
      return {
        bgColor: 'bg-yellow-50 dark:bg-yellow-900/50',
        borderColor: 'border-t border-t-yellow-400 dark:border-t-yellow-700',
        textColor: 'text-yellow-700 dark:text-yellow-400',
        icon: 'schedule',
        iconColor: 'text-yellow-500 dark:text-yellow-400',
      };
    default:
      return {
        bgColor: 'bg-gray-50 dark:bg-gray-900/50',
        borderColor: 'border-t border-t-gray-400 dark:border-t-gray-700',
        textColor: 'text-gray-700 dark:text-gray-400',
        icon: 'help',
        iconColor: 'text-gray-500 dark:text-gray-400',
      };
  }
};