/* @flow */
import * as React from 'react';

function safeSortObject(value: any, seen: WeakSet<any>): any {
  // return non-object value as is
  if (value === null || typeof value !== 'object') {
    return value;
  }

  // return date, regexp and react element values as is
  if (
    value instanceof Date ||
    value instanceof RegExp ||
    React.isValidElement(value)
  ) {
    return value;
  }

  seen.add(value);

  // make a copy of array with each item passed through the sorting algorithm
  if (Array.isArray(value)) {
    return value.map(v => safeSortObject(v, seen));
  }

  // make a copy of object with key sorted
  return Object.keys(value)
    .sort()
    .reduce((result, key) => {
      if (key === '_owner') {
        return result;
      }
      if (key === 'current' || seen.has(value[key])) {
        // eslint-disable-next-line no-param-reassign
        result[key] = '[Circular]';
      } else {
        // eslint-disable-next-line no-param-reassign
        result[key] = safeSortObject(value[key], seen);
      }
      return result;
    }, {});
}

export default function sortObject(value: any): any {
  return safeSortObject(value, new WeakSet());
}
