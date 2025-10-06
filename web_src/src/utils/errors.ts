import { AlertAlertType, SuperplaneAlert } from "@/api-client";
import { ErrorType, TooltipError } from "@/components/Tooltip/errors-tooltip";

const ALERT_TYPE_TO_ERROR_TYPE: Record<AlertAlertType, ErrorType> = {
  'ALERT_TYPE_ERROR': ErrorType.ERROR,
  'ALERT_TYPE_WARNING': ErrorType.WARNING,
  'ALERT_TYPE_INFO': ErrorType.INFO,
  'ALERT_TYPE_UNKNOWN': ErrorType.INFO,
}

export function alertsToErrorTooltip(alerts: SuperplaneAlert[]): TooltipError[] {
  return alerts.map(alert => alertToErrorTooltip(alert));
}

export function alertToErrorTooltip(alert: SuperplaneAlert): TooltipError {
  return {
    id: alert?.id || '',
    type: ALERT_TYPE_TO_ERROR_TYPE[alert?.type || 'ALERT_TYPE_INFO'],
    message: alert?.message || '',
  };
}