/**
 * Functions for rendering forms with proper labels and placeholders.
 */

export function getDefaultEventType(sourceType: string): string {
  switch (sourceType) {
    case 'github':
      return 'push';
    case 'semaphore':
      return 'pipeline_done';
    default:
      return '';
  }
}

export function getDefaultFilterExpression(sourceType: string): string {
  switch (sourceType) {
    case 'github':
      return '$.ref == "refs/heads/main"';
    case 'semaphore':
      return '$.pipeline.state == "done"';
    default:
      return '';
  }
}

export function getResourceType(sourceType: string): string {
  switch (sourceType) {
    case 'github':
      return 'repository';
    case 'semaphore':
      return 'project';
    default:
      return '';
  }
}

export function getResourceLabel(sourceType: string): string {
  const type = getResourceType(sourceType);
  return type.charAt(0).toUpperCase() + type.slice(1);
}

export function getResourcePlaceholder(sourceType: string): string {
  switch (sourceType) {
    case 'github':
      return 'my-repository';
    case 'semaphore':
      return 'my-semaphore-project';
    default:
      return '';
  }
}

export function getIntegrationLabel(sourceType: string): string {
  switch (sourceType) {
    case 'github':
      return 'GitHub repository';
    case 'semaphore':
      return 'Semaphore project';
    default:
      return '';
  }
}

export function getEventTypePlaceholder(sourceType: string): string {
  switch (sourceType) {
    case 'github':
      return 'e.g., push, pull_request, deployment';
    case 'semaphore':
      return 'e.g., pipeline_done';
    default:
      return 'Event type';
  }
}


/*
 * Returns true if the source type is an event source that doesn't require an integration
 * (manual, scheduled, webhook).
 */
export function isRegularEventSource(sourceType: string): boolean {
  return sourceType === 'manual' || sourceType === 'scheduled' || sourceType === 'webhook';
}