import { D as DiffOptions } from './types-f5c02aaf.js';
import 'pretty-format';

declare function serializeError(val: any, seen?: WeakMap<object, any>): any;
declare function processError(err: any, diffOptions?: DiffOptions): any;
declare function replaceAsymmetricMatcher(actual: any, expected: any, actualReplaced?: WeakSet<object>, expectedReplaced?: WeakSet<object>): {
    replacedActual: any;
    replacedExpected: any;
};

export { processError, replaceAsymmetricMatcher, serializeError };
