export function matches(selector: string, node: object, opt?: {
    warn?: boolean;
}): boolean;
export function closest(selector: string, node: object, opt?: {
    warn?: boolean;
}): object | null;
export function querySelector(selector: string, node: object, opt?: {
    warn?: boolean;
}): object | null;
export function querySelectorAll(selector: string, node: object, opt?: {
    warn?: boolean;
}): Array<object | undefined>;
