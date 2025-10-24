/**
 * Functions for rendering forms with proper labels and placeholders.
 */

import type { ComponentsConfigurationField } from '../api-client'

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

/**
 * Checks if a field is visible based on its visibility conditions
 */
export function isFieldVisible(
  field: ComponentsConfigurationField,
  allValues: Record<string, any>
): boolean {
  if (!field.visibilityConditions || field.visibilityConditions.length === 0) {
    return true
  }

  // All conditions must be satisfied (AND logic)
  return field.visibilityConditions.every((condition) => {
    if (!condition.field || !condition.values) {
      return true
    }

    const fieldValue = allValues[condition.field]

    // Convert field value to string for comparison
    const fieldValueStr = fieldValue !== undefined && fieldValue !== null
      ? String(fieldValue)
      : ''

    // Check if the field value matches any of the expected values
    // Support wildcard "*" to match any non-empty value
    return condition.values.some((expectedValue) => {
      if (expectedValue === '*') {
        // Wildcard matches any non-empty value
        return fieldValueStr !== ''
      }
      return fieldValueStr === expectedValue
    })
  })
}

/**
 * Filters a configuration object to only include visible fields based on their visibility conditions.
 * This ensures that hidden fields are not included in the API payload.
 *
 * @param configuration - The full configuration object with all field values
 * @param fields - The configuration field definitions including visibility conditions
 * @returns A filtered configuration object containing only visible fields
 */
export function filterVisibleConfiguration(
  configuration: Record<string, any>,
  fields: ComponentsConfigurationField[]
): Record<string, any> {
  const filtered: Record<string, any> = {}

  for (const field of fields) {
    if (field.name && isFieldVisible(field, configuration)) {
      // Only include the field if it's visible and has a value
      if (configuration[field.name] !== undefined) {
        filtered[field.name] = configuration[field.name]
      }
    }
  }

  return filtered
}