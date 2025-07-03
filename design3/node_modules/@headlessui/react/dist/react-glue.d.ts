import { shallowEqual, type Machine } from './machine.js';
export declare function useSlice<M extends Machine<any, any>, Slice>(machine: M, selector: (state: Readonly<M extends Machine<infer State, any> ? State : never>) => Slice, compare?: typeof shallowEqual): Slice;
