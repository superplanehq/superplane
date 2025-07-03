export function unescapeSelector(selector?: string): string | null;
export function preprocess(...args: any[]): string;
export function parseSelector(selector: string): object;
export function walkAST(ast?: object): Array<object | undefined>;
export { generate as generateCSS } from "css-tree";
