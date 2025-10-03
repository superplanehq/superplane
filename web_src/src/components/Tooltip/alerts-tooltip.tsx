import Tippy from '@tippyjs/react/headless';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { SuperplaneAlert, AlertAlertType } from '@/api-client';
import { useState } from 'react';

interface AlertsTooltipProps {
  alerts: SuperplaneAlert[];
  onAcknowledge: (alertId: string) => void;
  className?: string;
  isLoading?: boolean;
}

export function AlertsTooltip({ alerts, onAcknowledge, className = '', isLoading = false }: AlertsTooltipProps) {
  const [hoveredGroupIndex, setHoveredGroupIndex] = useState<number | null>(null);
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

  console.log(alerts)

  const getIconColor = (type: AlertAlertType) => {
    switch (type) {
      case 'ALERT_TYPE_ERROR':
        return 'text-red-600 dark:text-red-500';
      case 'ALERT_TYPE_WARNING':
        return 'text-yellow-600 dark:text-yellow-500';
      case 'ALERT_TYPE_INFO':
        return 'text-blue-600 dark:text-blue-500';
      default:
        return 'text-blue-600 dark:text-blue-500';
    }
  };

  const groupedAlerts = alerts.reduce((acc, alert) => {
    const key = `${alert.type}-${alert.message}`;
    if (!acc[key]) {
      acc[key] = {
        type: alert.type || 'ALERT_TYPE_INFO',
        message: alert.message || 'No message',
        count: 0,
        alertIds: []
      };
    }
    acc[key].count++;
    if (alert.id) {
      acc[key].alertIds.push(alert.id);
    }
    return acc;
  }, {} as Record<string, { type: AlertAlertType; message: string; count: number; alertIds: string[] }>);

  const sortedGroups = Object.values(groupedAlerts).sort((a, b) => {
    const typeOrder = { 'ALERT_TYPE_ERROR': 0, 'ALERT_TYPE_WARNING': 1, 'ALERT_TYPE_INFO': 2 };
    return (typeOrder[a.type as keyof typeof typeOrder] || 3) - (typeOrder[b.type as keyof typeof typeOrder] || 3);
  });

  const getTypeLabel = (type: AlertAlertType, count: number) => {
    const labels = {
      'ALERT_TYPE_ERROR': count === 1 ? 'error' : 'errors',
      'ALERT_TYPE_WARNING': count === 1 ? 'warning' : 'warnings',
      'ALERT_TYPE_INFO': count === 1 ? 'info alert' : 'info alerts',
      'ALERT_TYPE_UNKNOWN': count === 1 ? 'alert' : 'alerts',
    };
    return labels[type] || 'alerts';
  };

  const getBackgroundColor = (type: AlertAlertType) => {
    switch (type) {
      case 'ALERT_TYPE_ERROR':
        return 'bg-red-50 dark:bg-red-900/20';
      case 'ALERT_TYPE_WARNING':
        return 'bg-yellow-50 dark:bg-yellow-900/20';
      case 'ALERT_TYPE_INFO':
        return 'bg-blue-50 dark:bg-blue-900/20';
      default:
        return 'bg-blue-50 dark:bg-blue-900/20';
    }
  };


  const alertCounts = {
    error: alerts.filter(alert => alert.type === 'ALERT_TYPE_ERROR').length,
    warning: alerts.filter(alert => alert.type === 'ALERT_TYPE_WARNING').length,
    info: alerts.filter(alert => alert.type === 'ALERT_TYPE_INFO').length
  };

  const dominantAlertType = (() => {
    if (alertCounts.error > 0) return 'ALERT_TYPE_ERROR';
    if (alertCounts.warning > 0) return 'ALERT_TYPE_WARNING';
    return 'ALERT_TYPE_INFO';
  })();

  if (alerts.length === 0 && !isLoading) {
    return null;
  }

  if (isLoading) {
    return (
      <div
        className={`transition-colors cursor-pointer relative ${className}`}
        role="button"
        tabIndex={0}
      >
        <div className="flex items-center gap-1 px-2 py-1 rounded-md bg-gray-50 dark:bg-gray-800">
          <MaterialSymbol name="sync" size="sm" className="text-gray-500 dark:text-gray-400 animate-spin" fill={1} />
        </div>
      </div>
    );
  }

  return (
    <Tippy
      render={() => (
        <div className="p-3 min-w-[350px] bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg">
          <div className="text-left font-medium text-sm mb-3 text-gray-900 dark:text-gray-100">
            Alerts
          </div>
          <div className="space-y-2 flex flex-col gap-6">
            {sortedGroups.map((group, index) => (
              <div
                key={index}
                className={`text-xs cursor-pointer p-2 -m-2 rounded transition-colors ${getBackgroundColor(group.type)}`}
                onMouseEnter={() => setHoveredGroupIndex(index)}
                onMouseLeave={() => setHoveredGroupIndex(null)}
                onClick={async (e) => {
                  e.stopPropagation()
                  await Promise.all(group.alertIds.map(id => onAcknowledge(id)));
                }}
              >
                <div className="flex items-center gap-2">
                  <MaterialSymbol name={getIconName(group.type)} size="sm" className={`${getIconColor(group.type)} flex-shrink-0`} fill={1} />
                  <div className="flex-1 min-w-0 text-left">
                    <span className={`font-medium ${getIconColor(group.type)}`}>
                      {group.count} {getTypeLabel(group.type, group.count)}:
                    </span>
                    <span className="text-gray-600 dark:text-gray-400 ml-1 break-words">
                      {group.message}
                    </span>
                  </div>
                  <div className="w-6 flex justify-center">
                    {hoveredGroupIndex === index && (
                      <MaterialSymbol
                        name="close"
                        size="sm"
                        className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 flex-shrink-0"
                        fill={1}
                      />
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-600">
            <button
              onClick={async (e) => {
                e.stopPropagation()
                const allAlertIds = sortedGroups.flatMap(group => group.alertIds);
                await Promise.all(allAlertIds.map(id => onAcknowledge(id)));
              }}
              className="text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 text-xs underline"
            >
              Acknowledge all →
            </button>
          </div>
        </div>
      )}
      placement="bottom-end"
      interactive={true}
      delay={200}
    >
      <div
        className={`transition-colors cursor-pointer relative ${className}`}
        role="button"
        tabIndex={0}
      >
        <div className={`flex items-center gap-1 px-2 py-1 rounded-md ${getBackgroundColor(dominantAlertType)}`}>
          {alertCounts.error > 0 && (
            <div className="flex items-center gap-0.5">
              <MaterialSymbol name="error" size="sm" className="text-red-600 dark:text-red-500" fill={1} />
              <span className="text-xs font-medium text-red-600 dark:text-red-500">{alertCounts.error}</span>
            </div>
          )}
          {alertCounts.warning > 0 && (
            <div className="flex items-center gap-0.5">
              <MaterialSymbol name="warning" size="sm" className="text-yellow-600 dark:text-yellow-500" fill={1} />
              <span className="text-xs font-medium text-yellow-600 dark:text-yellow-500">{alertCounts.warning}</span>
            </div>
          )}
          {alertCounts.info > 0 && (
            <div className="flex items-center gap-0.5">
              <MaterialSymbol name="info" size="sm" className="text-blue-600 dark:text-blue-500" fill={1} />
              <span className="text-xs font-medium text-blue-600 dark:text-blue-500">{alertCounts.info}</span>
            </div>
          )}
        </div>
      </div>
    </Tippy>
  );
}