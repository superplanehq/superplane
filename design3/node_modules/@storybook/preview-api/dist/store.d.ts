export { S as StoryStore, c as prepareContext, a as prepareMeta, p as prepareStory, b as processCSFFile } from './StoryStore-f7424ddf.js';
export { D as DEEPLY_EQUAL, P as PropDescriptor, U as UNTARGETED, c as combineArgs, a as combineParameters, b as composeStepRunners, d as composeStories, e as composeStory, f as decorateStory, o as deepDiff, g as defaultDecorateStory, h as filterArgTypes, p as groupArgsByTarget, j as inferControls, m as mapArgsToTypes, q as noTargetArgs, n as normalizeStory, s as sanitizeStoryContextUpdate, i as setProjectAnnotations, r as sortStoriesV6, l as sortStoriesV7, k as userOrAutoTitle, u as userOrAutoTitleFromSpecifier, v as validateOptions } from './sortStories-436f1964.js';
import * as _storybook_types from '@storybook/types';
import { InputType, StrictInputType, ArgTypes, GlobalTypes, StrictArgTypes, StrictGlobalTypes, Renderer, ModuleExports, NormalizedComponentAnnotations, ProjectAnnotations, NormalizedProjectAnnotations } from '@storybook/types';
export { c as composeConfigs, a as getArrayField, g as getField, b as getObjectField, d as getSingletonField } from './composeConfigs-62a04721.js';
export { H as HooksContext, k as applyHooks, u as useArgs, a as useCallback, b as useChannel, c as useEffect, d as useGlobals, e as useMemo, f as useParameter, g as useReducer, h as useRef, i as useState, j as useStoryContext } from './hooks-655fa363.js';
import 'synchronous-promise';

declare const normalizeInputType: (inputType: InputType, key: string) => StrictInputType;
declare const normalizeInputTypes: (inputTypes: ArgTypes | GlobalTypes) => StrictArgTypes | StrictGlobalTypes;

declare function normalizeComponentAnnotations<TRenderer extends Renderer>(defaultExport: ModuleExports['default'], title?: string, importPath?: string): NormalizedComponentAnnotations<TRenderer>;

declare function normalizeProjectAnnotations<TRenderer extends Renderer>({ argTypes, globalTypes, argTypesEnhancers, decorators, loaders, ...annotations }: ProjectAnnotations<TRenderer>): NormalizedProjectAnnotations<TRenderer>;

declare const getValuesFromArgTypes: (argTypes?: ArgTypes) => ArgTypes<_storybook_types.Args>;

/**
 * @param {string} sharedId - The ID of the shared state.
 * @param {S} [defaultState] - The default state of the shared state.
 * @deprecated This API might get dropped, if you are using this, please file an issue.
 * @returns {[S, (s: S) => void]} - A tuple containing the current state and a function to update the state.
 * @example
 * const [state, setState] = useSharedState('my-addon', { count: 0 });
 * console.log(state); // { count: 0 }
 * setState({ count: 1 });
 * console.log(state); // { count: 1 }
 */
declare function useSharedState<S>(sharedId: string, defaultState?: S): [S, (s: S) => void];
/**
 * @param {string} sharedId - The ID of the shared state.
 * @param {S} [defaultState] - The default state of the shared state.
 * @deprecated This API might get dropped, if you are using this, please file an issue.
 * @returns {[S, (s: S) => void]} - A tuple containing the current state and a function to update the state.
 * @example
 * const [state, setState] = useSharedState('my-addon', { count: 0 });
 * console.log(state); // { count: 0 }
 * setState({ count: 1 });
 * console.log(state); // { count: 1 }
 */
declare function useAddonState<S>(addonId: string, defaultState?: S): [S, (s: S) => void];

export { getValuesFromArgTypes, normalizeComponentAnnotations, normalizeInputType, normalizeInputTypes, normalizeProjectAnnotations, useAddonState, useSharedState };
