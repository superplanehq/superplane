import type { Declaration } from 'postcss';
declare const tanFunctionCheck = "tan(";
declare function transformTanFunction(decl: Declaration): string | undefined;
export { tanFunctionCheck, transformTanFunction };
