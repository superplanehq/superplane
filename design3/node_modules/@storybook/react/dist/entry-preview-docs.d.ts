import * as _storybook_docs_tools from '@storybook/docs-tools';
import { extractComponentDescription } from '@storybook/docs-tools';
import { LegacyStoryFn, DecoratorFunction, ArgTypesEnhancer } from '@storybook/types';
import { R as ReactRenderer } from './types-0fc72a6d.js';
import 'react';

declare const applyDecorators: (storyFn: LegacyStoryFn<ReactRenderer>, decorators: DecoratorFunction<ReactRenderer>[]) => LegacyStoryFn<ReactRenderer>;

declare const parameters: {
    docs: {
        story: {
            inline: boolean;
        };
        extractArgTypes: _storybook_docs_tools.ArgTypesExtractor;
        extractComponentDescription: typeof extractComponentDescription;
    };
};
declare const decorators: DecoratorFunction<ReactRenderer>[];
declare const argTypesEnhancers: ArgTypesEnhancer<ReactRenderer>[];

export { applyDecorators, argTypesEnhancers, decorators, parameters };
