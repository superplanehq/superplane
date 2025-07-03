import { Indexer, StorybookConfig, Options } from '@storybook/types';

declare const createStoriesMdxIndexer: (legacyMdx1?: boolean) => Indexer;
declare const addons: StorybookConfig['addons'];
declare const viteFinal: (config: any, options: Options) => Promise<any>;
declare const webpackX: any;
declare const indexersX: any;
declare const docsX: any;
declare const optimizeViteDeps: string[];

export { addons, createStoriesMdxIndexer, docsX as docs, indexersX as experimental_indexers, optimizeViteDeps, viteFinal, webpackX as webpack };
