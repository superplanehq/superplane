import type { Classes } from './types';
/**
 * Combines an array of classes into a single space delimited-string.
 *
 * @param classes List of input classes. Values are filtered out if they evaluate to `false`. See {@link Boolean}.
 */
export declare function combineClasses(...classes: Classes): string;
