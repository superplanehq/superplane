import { ModuleExports, Renderer, ProjectAnnotations } from '@storybook/types';

declare function getField<TFieldType = any>(moduleExportList: ModuleExports[], field: string): TFieldType | TFieldType[];
declare function getArrayField<TFieldType = any>(moduleExportList: ModuleExports[], field: string, options?: {
    reverseFileOrder?: boolean;
}): TFieldType[];
declare function getObjectField<TFieldType = Record<string, any>>(moduleExportList: ModuleExports[], field: string): TFieldType;
declare function getSingletonField<TFieldType = any>(moduleExportList: ModuleExports[], field: string): TFieldType;
declare function composeConfigs<TRenderer extends Renderer>(moduleExportList: ModuleExports[]): ProjectAnnotations<TRenderer>;

export { getArrayField as a, getObjectField as b, composeConfigs as c, getSingletonField as d, getField as g };
