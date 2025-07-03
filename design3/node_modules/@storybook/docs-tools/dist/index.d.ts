import * as _storybook_types from '@storybook/types';
import { StrictArgTypes, Renderer, StoryContextForEnhancers, Options } from '@storybook/types';

interface JsDocParam {
    name: string;
    description?: string;
}
interface JsDocReturns {
    description?: string;
}
interface JsDocTags {
    params?: JsDocParam[];
    returns?: JsDocReturns;
}
interface PropSummaryValue {
    summary: string;
    detail?: string;
}
type PropType = PropSummaryValue;
type PropDefaultValue = PropSummaryValue;
interface PropDef {
    name: string;
    type: PropType;
    sbType?: any;
    required: boolean;
    description?: string;
    defaultValue?: PropDefaultValue;
    jsDocTags?: JsDocTags;
}

type Component = any;

type PropsExtractor = (component: Component) => {
    rows?: PropDef[];
} | null;
type ArgTypesExtractor = (component: Component) => StrictArgTypes | null;
interface DocgenType {
    name: string;
    description?: string;
    required?: boolean;
    value?: any;
}
interface DocgenPropType extends DocgenType {
    value?: any;
    raw?: string;
    computed?: boolean;
}
interface DocgenFlowType extends DocgenType {
    type?: string;
    raw?: string;
    signature?: any;
    elements?: any[];
}
interface DocgenTypeScriptType extends DocgenType {
    raw?: string;
}
interface DocgenPropDefaultValue {
    value: string;
    computed?: boolean;
    func?: boolean;
}
interface DocgenInfo {
    type?: DocgenPropType;
    flowType?: DocgenFlowType;
    tsType?: DocgenTypeScriptType;
    required: boolean;
    description?: string;
    defaultValue?: DocgenPropDefaultValue;
}
declare enum TypeSystem {
    JAVASCRIPT = "JavaScript",
    FLOW = "Flow",
    TYPESCRIPT = "TypeScript",
    UNKNOWN = "Unknown"
}

declare const convert: (docgenInfo: DocgenInfo) => any;

declare function isDefaultValueBlacklisted(value: string): boolean;

declare const str: (obj: any) => string;

declare function hasDocgen(component: Component): boolean;
declare function isValidDocgenSection(docgenSection: any): boolean;
declare function getDocgenSection(component: Component, section: string): any;
declare function getDocgenDescription(component: Component): string;

interface ExtractedJsDocParam {
    name: string;
    type?: any;
    description?: string;
    getPrettyName: () => string;
    getTypeName: () => string;
}
interface ExtractedJsDocReturns {
    type?: any;
    description?: string;
    getTypeName: () => string;
}
interface ExtractedJsDoc {
    params?: ExtractedJsDocParam[];
    deprecated?: string;
    returns?: ExtractedJsDocReturns;
    ignore: boolean;
}
interface JsDocParsingOptions {
    tags?: string[];
}
interface JsDocParsingResult {
    includesJsDoc: boolean;
    ignore: boolean;
    description?: string;
    extractedTags?: ExtractedJsDoc;
}
type ParseJsDoc = (value?: string, options?: JsDocParsingOptions) => JsDocParsingResult;
declare const parseJsDoc: ParseJsDoc;

interface ExtractedProp {
    propDef: PropDef;
    docgenInfo: DocgenInfo;
    jsDocTags: ExtractedJsDoc;
    typeSystem: TypeSystem;
}
type ExtractProps = (component: Component, section: string) => ExtractedProp[];
declare const extractComponentSectionArray: (docgenSection: any) => any;
declare const extractComponentSectionObject: (docgenSection: any) => ExtractedProp[];
declare const extractComponentProps: ExtractProps;
declare function extractComponentDescription(component?: Component): string;

declare const MAX_TYPE_SUMMARY_LENGTH = 90;
declare const MAX_DEFAULT_VALUE_SUMMARY_LENGTH = 50;
declare function isTooLongForTypeSummary(value: string): boolean;
declare function isTooLongForDefaultValueSummary(value: string): boolean;
declare function createSummaryValue(summary: string, detail?: string): PropSummaryValue;
declare const normalizeNewlines: (string: string) => string;

declare const enhanceArgTypes: <TRenderer extends Renderer>(context: StoryContextForEnhancers<TRenderer, _storybook_types.Args>) => _storybook_types.StrictArgTypes<_storybook_types.Args>;

declare const ADDON_ID = "storybook/docs";
declare const PANEL_ID = "storybook/docs/panel";
declare const PARAM_KEY = "docs";
declare const SNIPPET_RENDERED = "storybook/docs/snippet-rendered";
declare enum SourceType {
    /**
     * AUTO is the default
     *
     * Use the CODE logic if:
     * - the user has set a custom source snippet in `docs.source.code` story parameter
     * - the story is not an args-based story
     *
     * Use the DYNAMIC rendered snippet if the story is an args story
     */
    AUTO = "auto",
    /**
     * Render the code extracted by source-loader
     */
    CODE = "code",
    /**
     * Render dynamically-rendered source snippet from the story's virtual DOM (currently React only)
     */
    DYNAMIC = "dynamic"
}

declare const hasDocsOrControls: (options: Options) => boolean;

export { ADDON_ID, ArgTypesExtractor, Component, DocgenFlowType, DocgenInfo, DocgenPropDefaultValue, DocgenPropType, DocgenType, DocgenTypeScriptType, ExtractProps, ExtractedJsDoc, ExtractedJsDocParam, ExtractedJsDocReturns, ExtractedProp, JsDocParam, JsDocParsingOptions, JsDocParsingResult, JsDocReturns, JsDocTags, MAX_DEFAULT_VALUE_SUMMARY_LENGTH, MAX_TYPE_SUMMARY_LENGTH, PANEL_ID, PARAM_KEY, ParseJsDoc, PropDef, PropDefaultValue, PropSummaryValue, PropType, PropsExtractor, SNIPPET_RENDERED, SourceType, TypeSystem, convert, createSummaryValue, enhanceArgTypes, extractComponentDescription, extractComponentProps, extractComponentSectionArray, extractComponentSectionObject, getDocgenDescription, getDocgenSection, hasDocgen, hasDocsOrControls, isDefaultValueBlacklisted, isTooLongForDefaultValueSummary, isTooLongForTypeSummary, isValidDocgenSection, normalizeNewlines, parseJsDoc, str };
