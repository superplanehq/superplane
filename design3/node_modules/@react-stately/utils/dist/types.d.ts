export function useControlledState<T, C = T>(value: Exclude<T, undefined>, defaultValue: Exclude<T, undefined> | undefined, onChange?: (v: C, ...args: any[]) => void): [T, (value: T, ...args: any[]) => void];
export function useControlledState<T, C = T>(value: Exclude<T, undefined> | undefined, defaultValue: Exclude<T, undefined>, onChange?: (v: C, ...args: any[]) => void): [T, (value: T, ...args: any[]) => void];
/**
 * Takes a value and forces it to the closest min/max if it's outside. Also forces it to the closest valid step.
 */
export function clamp(value: number, min?: number, max?: number): number;
export function snapValueToStep(value: number, min: number | undefined, max: number | undefined, step: number): number;
export function toFixedNumber(value: number, digits: number, base?: number): number;

//# sourceMappingURL=types.d.ts.map
