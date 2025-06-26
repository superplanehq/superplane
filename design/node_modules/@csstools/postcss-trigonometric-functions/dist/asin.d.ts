import type { Declaration } from 'postcss';
declare const asinFunctionCheck = "asin(";
declare function transformAsinFunction(decl: Declaration): string | undefined;
export { asinFunctionCheck, transformAsinFunction };
