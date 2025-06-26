import type { Declaration } from 'postcss';
declare const cosFunctionCheck = "cos(";
declare function transformCosFunction(decl: Declaration): string | undefined;
export { cosFunctionCheck, transformCosFunction };
