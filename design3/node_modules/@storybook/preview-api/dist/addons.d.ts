import { Channel } from '@storybook/channels';
export { E as EventMap, H as HooksContext, L as Listener, k as applyHooks, u as useArgs, a as useCallback, b as useChannel, c as useEffect, d as useGlobals, e as useMemo, f as useParameter, g as useReducer, h as useRef, i as useState, j as useStoryContext } from './hooks-655fa363.js';
import { Addon_StoryWrapper } from '@storybook/types';

declare class AddonStore {
    constructor();
    private channel;
    /**
     * @deprecated will be removed in 8.0, please use channel instead
     */
    private serverChannel;
    private promise;
    private resolve;
    getChannel: () => Channel;
    /**
     * @deprecated will be removed in 8.0, please use getChannel instead
     */
    getServerChannel: () => Channel;
    ready: () => Promise<Channel>;
    hasChannel: () => boolean;
    /**
     * @deprecated will be removed in 8.0, please use the normal channel instead
     */
    hasServerChannel: () => boolean;
    setChannel: (channel: Channel) => void;
    /**
     * @deprecated will be removed in 8.0, please use the normal channel instead
     */
    setServerChannel: (channel: Channel) => void;
}
declare const addons: AddonStore;

type MakeDecoratorResult = (...args: any) => any;
interface MakeDecoratorOptions {
    name: string;
    parameterName: string;
    skipIfNoParametersOrOptions?: boolean;
    wrapper: Addon_StoryWrapper;
}
/**
 * Creates a Storybook decorator function that can be used to wrap stories with additional functionality.
 *
 * @param {MakeDecoratorOptions} options - The options for the decorator.
 * @param {string} options.name - The name of the decorator.
 * @param {string} options.parameterName - The name of the parameter that will be used to pass options to the decorator.
 * @param {Addon_StoryWrapper} options.wrapper - The function that will be used to wrap the story.
 * @param {boolean} [options.skipIfNoParametersOrOptions=false] - Whether to skip the decorator if no options or parameters are provided.
 * @returns {MakeDecoratorResult} A function that can be used as a Storybook decorator.
 *
 * @example
 * const myDecorator = makeDecorator({
 *   name: 'My Decorator',
 *   parameterName: 'myDecorator',
 *   wrapper: (storyFn, context, { options }) => {
 *     const { myOption } = options;
 *     return <div style={{ backgroundColor: myOption }}>{storyFn()}</div>;
 *   },
 * });
 *
 * export const decorators = [myDecorator];
 */
declare const makeDecorator: ({ name, parameterName, wrapper, skipIfNoParametersOrOptions, }: MakeDecoratorOptions) => MakeDecoratorResult;

declare function mockChannel(): Channel;

export { AddonStore, MakeDecoratorOptions, MakeDecoratorResult, addons, makeDecorator, mockChannel };
