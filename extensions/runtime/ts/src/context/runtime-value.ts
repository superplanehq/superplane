export type RuntimeValue =
  | null
  | boolean
  | number
  | string
  | RuntimeValue[]
  | { [key: string]: RuntimeValue };
