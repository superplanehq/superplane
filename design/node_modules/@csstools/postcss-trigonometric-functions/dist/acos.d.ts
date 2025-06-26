import type { Declaration } from 'postcss';
declare const acosFunctionCheck = "acos(";
declare function transformAcosFunction(decl: Declaration): string | undefined;
export { acosFunctionCheck, transformAcosFunction };
