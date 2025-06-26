import type { Declaration } from 'postcss';
declare const atan2FunctionCheck = "atan2(";
declare function transformAtan2Function(decl: Declaration): string | undefined;
export { atan2FunctionCheck, transformAtan2Function };
