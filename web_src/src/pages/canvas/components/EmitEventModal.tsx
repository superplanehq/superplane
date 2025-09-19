import { useState, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { Dialog, DialogTitle, DialogDescription, DialogBody, DialogActions } from '@/components/Dialog/dialog';
import { Input } from '@/components/Input/input';
import { Textarea } from '@/components/Textarea/textarea';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Badge } from '@/components/Badge/badge';
import { Button } from '@/components/Button/button';
import { superplaneCreateEvent, SuperplaneEventSourceType, SuperplaneEvent } from '@/api-client';
import { useCanvasStore } from '../store/canvasStore';
import { withOrganizationHeader } from '@/utils/withOrganizationHeader';

interface EmitEventModalProps {
  isOpen: boolean;
  onClose: () => void;
  sourceId: string;
  sourceName: string;
  sourceType: 'event_source' | 'stage' | 'connection_group';
  lastEvent?: SuperplaneEvent;
}

// Map our internal source type to API enum
const getApiSourceType = (sourceType: string): SuperplaneEventSourceType => {
  switch (sourceType) {
    case 'event_source':
      return 'EVENT_SOURCE_TYPE_EVENT_SOURCE';
    case 'stage':
      return 'EVENT_SOURCE_TYPE_STAGE';
    case 'connection_group':
      return 'EVENT_SOURCE_TYPE_CONNECTION_GROUP';
    default:
      return 'EVENT_SOURCE_TYPE_UNKNOWN';
  }
};

const getSourceTypeLabel = (sourceType: string): string => {
  switch (sourceType) {
    case 'event_source':
      return 'Event Source';
    case 'stage':
      return 'Stage';
    case 'connection_group':
      return 'Connection Group';
    default:
      return 'Source';
  }
};

const getSourceIcon = (sourceType: string): string => {
  switch (sourceType) {
    case 'event_source':
      return 'webhook';
    case 'stage':
      return 'rocket_launch';
    case 'connection_group':
      return 'hub';
    default:
      return 'source';
  }
};

export function EmitEventModal({ isOpen, onClose, sourceId, sourceName, sourceType, lastEvent }: EmitEventModalProps) {
  const [eventType, setEventType] = useState('');
  const [rawEventData, setRawEventData] = useState('{}');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [jsonError, setJsonError] = useState<string | null>(null);
  const canvasId = useCanvasStore(state => state.canvasId);

  // Set default values from lastEvent when modal opens
  useEffect(() => {
    if (isOpen && lastEvent) {
      setEventType(lastEvent.type || '');
      if (lastEvent.raw) {
        setRawEventData(JSON.stringify(lastEvent.raw, null, 2));
      }
    }
  }, [isOpen, lastEvent]);

  const validateAndParseJson = (jsonString: string): { isValid: boolean; parsed?: any } => {
    try {
      const parsed = JSON.parse(jsonString);
      setJsonError(null);
      return { isValid: true, parsed };
    } catch (error) {
      setJsonError('Invalid JSON format');
      return { isValid: false };
    }
  };

  const handleRawDataChange = (value: string) => {
    setRawEventData(value);
    if (value.trim()) {
      validateAndParseJson(value);
    } else {
      setJsonError(null);
    }
  };

  const handleSubmit = async () => {
    if (!eventType.trim()) {
      setError('Event type is required');
      return;
    }

    const { isValid, parsed } = validateAndParseJson(rawEventData);
    if (!isValid) {
      setError('Please fix the JSON format');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      await superplaneCreateEvent(withOrganizationHeader({
        path: { canvasIdOrName: canvasId! },
        body: {
          sourceType: getApiSourceType(sourceType),
          sourceId: sourceId,
          type: eventType,
          raw: parsed
        }
      }));

      // Reset form and close modal
      setEventType('');
      setRawEventData('{}');
      setError(null);
      setJsonError(null);
      onClose();
    } catch (error: any) {
      setError(error?.message || 'Failed to emit event');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleClose = () => {
    if (!isSubmitting) {
      setEventType('');
      setRawEventData('{}');
      setError(null);
      setJsonError(null);
      onClose();
    }
  };

  if (!isOpen) return null;

  return createPortal(
    <Dialog
      open={isOpen}
      onClose={handleClose}
      className="relative z-50"
      size="lg"
    >
      <DialogTitle className="flex items-center gap-3">
        <MaterialSymbol name={getSourceIcon(sourceType)} size="lg" />
        <div>
          <div className="text-xl font-semibold">
            Emit Event
          </div>
          <div className="flex items-center gap-2 mt-1">
            <Badge color="zinc">
              {getSourceTypeLabel(sourceType)}
            </Badge>
            <span className="text-sm text-gray-600 dark:text-gray-400">
              {sourceName}
            </span>
          </div>
        </div>
      </DialogTitle>

      <DialogDescription>
        Manually emit a test event from this {getSourceTypeLabel(sourceType).toLowerCase()}
      </DialogDescription>

      <DialogBody>
        <div className="space-y-4">
          <div>
            <label htmlFor="eventType" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Event Type
            </label>
            <Input
              id="eventType"
              value={eventType}
              onChange={(e) => setEventType(e.target.value)}
              placeholder="e.g., webhook, push, deployment_complete"
              disabled={isSubmitting}
              className="w-full"
            />
          </div>

          <div>
            <label htmlFor="rawData" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Event Data (JSON)
            </label>
            <Textarea
              id="rawData"
              value={rawEventData}
              onChange={(e) => handleRawDataChange(e.target.value)}
              placeholder='{"key": "value"}'
              disabled={isSubmitting}
              className="w-full h-64 font-mono text-sm"
              rows={12}
            />
            {jsonError && (
              <p className="text-red-600 dark:text-red-400 text-sm mt-1">
                {jsonError}
              </p>
            )}
          </div>

          {error && (
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md p-3">
              <div className="flex items-center">
                <MaterialSymbol name="error" className="text-red-400 mr-2" />
                <span className="text-red-800 dark:text-red-200 text-sm">{error}</span>
              </div>
            </div>
          )}
        </div>
      </DialogBody>

      <DialogActions>
        <Button onClick={handleClose} disabled={isSubmitting}>
          Cancel
        </Button>
        <Button
          color="blue"
          onClick={handleSubmit}
          disabled={isSubmitting || !eventType.trim() || jsonError !== null}
        >
          {isSubmitting ? (
            <>
              <MaterialSymbol name="hourglass_empty" className="animate-spin" size="sm" />
              Emitting...
            </>
          ) : (
            'Emit Event'
          )}
        </Button>
      </DialogActions>
    </Dialog>,
    document.body
  );
}