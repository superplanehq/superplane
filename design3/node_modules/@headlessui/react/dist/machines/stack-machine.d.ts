import { Machine } from '../machine.js';
import { DefaultMap } from '../utils/default-map.js';
type Scope = string | null;
type Id = string;
interface State {
    stack: Id[];
}
export declare enum ActionTypes {
    Push = 0,
    Pop = 1
}
export type Actions = {
    type: ActionTypes.Push;
    id: Id;
} | {
    type: ActionTypes.Pop;
    id: Id;
};
declare class StackMachine extends Machine<State, Actions> {
    static new(): StackMachine;
    reduce(state: Readonly<State>, action: Actions): State;
    actions: {
        push: (id: Id) => void;
        pop: (id: Id) => void;
    };
    selectors: {
        isTop: (state: State, id: Id) => boolean;
        inStack: (state: State, id: Id) => boolean;
    };
}
export declare const stackMachines: DefaultMap<Scope, StackMachine>;
export {};
