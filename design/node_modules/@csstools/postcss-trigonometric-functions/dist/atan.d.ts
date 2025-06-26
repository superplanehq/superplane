import type { Declaration } from 'postcss';
declare const atanFunctionCheck = "atan(";
declare function transformAtanFunction(decl: Declaration): string | undefined;
export { atanFunctionCheck, transformAtanFunction };
