import Tippy from '@tippyjs/react/headless';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { SuperplaneAlert, AlertAlertType } from '@/api-client';

interface AlertsTooltipProps {
  alerts: SuperplaneAlert[];
  onAcknowledge: (alertId: string) => void;
  className?: string;
}

export function AlertsTooltip({ alerts, onAcknowledge, className = '' }: AlertsTooltipProps) {
  const getIconName = (type: AlertAlertType) => {
    switch (type) {
      case 'ALERT_TYPE_ERROR':
        return 'error';
      case 'ALERT_TYPE_WARNING':
        return 'warning';
      case 'ALERT_TYPE_INFO':
        return 'info';
      default:
        return 'info';
    }
  };

  const getIconColor = (type: AlertAlertType) => {
    switch (type) {
      case 'ALERT_TYPE_ERROR':
        return 'text-red-500';
      case 'ALERT_TYPE_WARNING':
        return 'text-yellow-500';
      case 'ALERT_TYPE_INFO':
        return 'text-blue-500';
      default:
        return 'text-blue-500';
    }
  };

  const getTriggerIconColor = () => {
    if (alerts.some(alert => alert.type === 'ALERT_TYPE_ERROR')) {
      return 'text-red-500 hover:text-red-600 dark:hover:text-red-400';
    }
    if (alerts.some(alert => alert.type === 'ALERT_TYPE_WARNING')) {
      return 'text-yellow-500 hover:text-yellow-600 dark:hover:text-yellow-400';
    }
    return 'text-blue-500 hover:text-blue-600 dark:hover:text-blue-400';
  };

  const getTriggerIcon = () => {
    if (alerts.some(alert => alert.type === 'ALERT_TYPE_ERROR')) return 'error';
    if (alerts.some(alert => alert.type === 'ALERT_TYPE_WARNING')) return 'warning';
    return 'info';
  };

  if (alerts.length === 0) {
    return null;
  }

  return (
    <Tippy
      render={() => (
        <div className="min-w-[300px] max-w-sm">
          <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 text-sm z-50">
            <div className="font-semibold mb-3 text-zinc-900 dark:text-zinc-100">Alerts</div>
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {alerts.map((alert) => (
                <div
                  key={alert.id}
                  className="flex items-start gap-2 p-2 bg-zinc-100 dark:bg-zinc-700 rounded border border-zinc-200 dark:border-zinc-600 hover:bg-zinc-200 dark:hover:bg-zinc-600 cursor-pointer transition-colors"
                  onClick={() => onAcknowledge(alert.id || '')}
                >
                  <MaterialSymbol
                    name={getIconName(alert.type || 'ALERT_TYPE_INFO')}
                    size="sm"
                    className={`${getIconColor(alert.type || 'ALERT_TYPE_INFO')} mt-0.5 flex-shrink-0`}
                  />
                  <span className="text-zinc-800 dark:text-zinc-200 text-sm flex-1">{alert.message || 'No message'}</span>
                  <MaterialSymbol
                    name="close"
                    size="sm"
                    className="text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200 flex-shrink-0"
                  />
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
      placement="bottom-end"
      interactive={true}
      delay={200}
    >
      <div
        className={`${getTriggerIconColor()} transition-colors cursor-pointer relative ${className}`}
        role="button"
        tabIndex={0}
      >
        <MaterialSymbol name={getTriggerIcon()} size="sm" />
        {alerts.length > 0 && (
          <span className="absolute -top-1 -right-1 bg-red-500 text-white text-xs rounded-full w-4 h-4 flex items-center justify-center">
            {alerts.length > 9 ? '9+' : alerts.length}
          </span>
        )}
      </div>
    </Tippy>
  );
}