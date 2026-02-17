import { GenericEntity } from '~/app/types/asset';

/**
 * Access a value from a generic entity by a dot-separated path.
 * For example, getFieldValue(entity, 'metadata.name') returns entity.metadata.name.
 */
export const getFieldValue = (entity: GenericEntity, path: string): unknown => {
  const parts = path.split('.');
  let current: unknown = entity;
  for (const part of parts) {
    if (current == null || typeof current !== 'object') {
      return undefined;
    }
    current = (current as Record<string, unknown>)[part];
  }
  return current;
};

/**
 * Render a field value as a displayable string.
 */
export const formatFieldValue = (value: unknown, type?: string): string => {
  if (value == null) {
    return '-';
  }
  if (type === 'tags' && Array.isArray(value)) {
    return value.join(', ');
  }
  if (type === 'boolean') {
    return value ? 'Yes' : 'No';
  }
  if (type === 'date' && typeof value === 'string') {
    try {
      return new Date(value).toLocaleDateString();
    } catch {
      return String(value);
    }
  }
  if (typeof value === 'object') {
    return JSON.stringify(value);
  }
  return String(value);
};
