import { Renderer, Args, StoryContext, StoryId, DecoratorApplicator } from '@storybook/types';

interface Hook {
    name: string;
    memoizedState?: any;
    deps?: any[] | undefined;
}
interface Effect {
    create: () => (() => void) | void;
    destroy?: (() => void) | void;
}
type AbstractFunction = (...args: any[]) => any;
declare class HooksContext<TRenderer extends Renderer, TArgs extends Args = Args> {
    hookListsMap: WeakMap<AbstractFunction, Hook[]>;
    mountedDecorators: Set<AbstractFunction>;
    prevMountedDecorators: Set<AbstractFunction>;
    currentHooks: Hook[];
    nextHookIndex: number;
    currentPhase: 'MOUNT' | 'UPDATE' | 'NONE';
    currentEffects: Effect[];
    prevEffects: Effect[];
    currentDecoratorName: string | null;
    hasUpdates: boolean;
    currentContext: StoryContext<TRenderer, TArgs> | null;
    renderListener: (storyId: StoryId) => void;
    constructor();
    init(): void;
    clean(): void;
    getNextHook(): Hook;
    triggerEffects(): void;
    addRenderListeners(): void;
    removeRenderListeners(): void;
}
declare const applyHooks: <TRenderer extends Renderer>(applyDecorators: DecoratorApplicator<TRenderer>) => DecoratorApplicator<TRenderer>;
/**
 * Returns a memoized value.
 * @template T The type of the memoized value.
 * @param {() => T} nextCreate A function that returns the memoized value.
 * @param {any[]} [deps] An optional array of dependencies. If any of the dependencies change, the memoized value will be recomputed.
 * @returns {T} The memoized value.
 * @example
 * const memoizedValue = useMemo(() => {
 *   return doExpensiveCalculation(a, b);
 * }, [a, b]);
 */
declare function useMemo<T>(nextCreate: () => T, deps?: any[]): T;
/** Returns a memoized callback.
 *
 * @template T The type of the callback function.
 * @param {T} callback The callback function to memoize.
 * @param {any[]} [deps] An optional array of dependencies. If any of the dependencies change, the memoized callback will be recomputed.
 * @returns {T} The memoized callback.
 *
 * @example
 * const memoizedCallback = useCallback(
 *   () => {
 *     doSomething(a, b);
 *   },
 *   [a, b],
 * );
 */
declare function useCallback<T>(callback: T, deps?: any[]): T;
/**
 * Returns a mutable ref object.
 *
 * @template T The type of the ref object.
 * @param {T} initialValue The initial value of the ref object.
 * @returns {{ current: T }} The mutable ref object.
 *
 * @example
 * const ref = useRef(0);
 * ref.current = 1;
 */
declare function useRef<T>(initialValue: T): {
    current: T;
};
/**
 * Returns a stateful value and a function to update it.
 *
 * @template S The type of the state.
 * @param {(() => S) | S} initialState The initial state value or a function that returns the initial state value.
 * @returns {[S, (update: ((prevState: S) => S) | S) => void]} An array containing the current state value and a function to update it.
 *
 * @example
 * const [count, setCount] = useState(0);
 * setCount(count + 1);
 */
declare function useState<S>(initialState: (() => S) | S): [S, (update: ((prevState: S) => S) | S) => void];
/**
 * A redux-like alternative to useState.
 *
 * @template S The type of the state.
 * @template A The type of the action.
 * @param {(state: S, action: A) => S} reducer The reducer function that returns the new state.
 * @param {S | I} initialArg The initial state value or the initial argument for the init function.
 * @param {(initialArg: I) => S} [init] An optional function that returns the initial state value.
 * @returns {[S, (action: A) => void]} An array containing the current state value and a function to dispatch actions.
 *
 * @example
 * const initialState = { count: 0 };
 *
 * function reducer(state, action) {
 *   switch (action.type) {
 *     case 'increment':
 *       return { count: state.count + 1 };
 *     case 'decrement':
 *       return { count: state.count - 1 };
 *     default:
 *       throw new Error();
 *   }
 * }
 *
 * function Counter() {
 *   const [state, dispatch] = useReducer(reducer, initialState);
 *   return (
 *     <>
 *       Count: {state.count}
 *       <button onClick={() => dispatch({ type: 'increment' })}>+</button>
 *       <button onClick={() => dispatch({ type: 'decrement' })}>-</button>
 *     </>
 *   );
 * }
 */
declare function useReducer<S, A>(reducer: (state: S, action: A) => S, initialState: S): [S, (action: A) => void];
declare function useReducer<S, I, A>(reducer: (state: S, action: A) => S, initialArg: I, init: (initialArg: I) => S): [S, (action: A) => void];
/**
 * Triggers a side effect, see https://reactjs.org/docs/hooks-reference.html#usestate
 * Effects are triggered synchronously after rendering the story
 *
 * @param {() => (() => void) | void} create A function that creates the effect. It should return a cleanup function, or nothing.
 * @param {any[]} [deps] An optional array of dependencies. If any of the dependencies change, the effect will be re-run.
 * @returns {void}
 *
 * @example
 * useEffect(() => {
 *   // Do something after rendering the story
 *   return () => {
 *     // Do something when the component unmounts or the effect is re-run
 *   };
 * }, [dependency1, dependency2]);
 */
declare function useEffect(create: () => (() => void) | void, deps?: any[]): void;
interface Listener {
    (...args: any[]): void;
}
interface EventMap {
    [eventId: string]: Listener;
}
/**
 * Subscribes to events emitted by the Storybook channel and returns a function to emit events.
 *
 * @param {EventMap} eventMap A map of event listeners to subscribe to.
 * @param {any[]} [deps=[]] An optional array of dependencies. If any of the dependencies change, the event listeners will be re-subscribed.
 * @returns {(...args: any[]) => void} A function to emit events to the Storybook channel.
 *
 * @example
 * // Subscribe to an event and emit it
 * const emit = useChannel({ 'my-event': (arg1, arg2) => console.log(arg1, arg2) });
 * emit('my-event', 'Hello', 'world!');
 */
declare function useChannel(eventMap: EventMap, deps?: any[]): (eventName: string, ...args: any) => void;
/**
 * Returns the current story context, including the story's ID, parameters, and other metadata.
 *
 * @template TRenderer The type of the story's renderer.
 * @template TArgs The type of the story's args.
 * @returns {StoryContext<TRenderer>} The current story context.
 *
 * @example
 * const { id, parameters } = useStoryContext();
 * console.log(`Current story ID: ${id}`);
 * console.log(`Current story parameters: ${JSON.stringify(parameters)}`);
 */
declare function useStoryContext<TRenderer extends Renderer, TArgs extends Args = Args>(): StoryContext<TRenderer>;
/**
 * Returns the value of a specific parameter for the current story, or a default value if the parameter is not set.
 *
 * @template S The type of the parameter value.
 * @param {string} parameterKey The key of the parameter to retrieve.
 * @param {S} [defaultValue] An optional default value to return if the parameter is not set.
 * @returns {S | undefined} The value of the parameter, or the default value if the parameter is not set.
 *
 * @example
 * // Retrieve the value of a parameter named "myParam"
 * const myParamValue = useParameter<string>('myParam', 'default value');
 * console.log(`The value of myParam is: ${myParamValue}`);
 */
declare function useParameter<S>(parameterKey: string, defaultValue?: S): S | undefined;
/**
 * Returns the current args for the story, and functions to update and reset them.
 *
 * @template TArgs The type of the story's args.
 * @returns {[TArgs, (newArgs: Partial<TArgs>) => void, (argNames?: (keyof TArgs)[]) => void]} An array containing the current args, a function to update them, and a function to reset them.
 *
 * @example
 * const [args, updateArgs, resetArgs] = useArgs<{ name: string, age: number }>();
 * console.log(`Current args: ${JSON.stringify(args)}`);
 * updateArgs({ name: 'John' });
 * resetArgs(['name']);
 */
declare function useArgs<TArgs extends Args = Args>(): [
    TArgs,
    (newArgs: Partial<TArgs>) => void,
    (argNames?: (keyof TArgs)[]) => void
];
/**
 * Returns the current global args for the story, and a function to update them.
 *
 * @returns {[Args, (newGlobals: Args) => void]} An array containing the current global args, and a function to update them.
 *
 * @example
 * const [globals, updateGlobals] = useGlobals();
 * console.log(`Current globals: ${JSON.stringify(globals)}`);
 * updateGlobals({ theme: 'dark' });
 */
declare function useGlobals(): [Args, (newGlobals: Args) => void];

export { EventMap as E, HooksContext as H, Listener as L, useCallback as a, useChannel as b, useEffect as c, useGlobals as d, useMemo as e, useParameter as f, useReducer as g, useRef as h, useState as i, useStoryContext as j, applyHooks as k, useArgs as u };
