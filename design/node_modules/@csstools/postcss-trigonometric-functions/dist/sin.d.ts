import type { Declaration } from 'postcss';
declare const sinFunctionCheck = "sin(";
declare function transformSinFunction(decl: Declaration): string | undefined;
export { sinFunctionCheck, transformSinFunction };
