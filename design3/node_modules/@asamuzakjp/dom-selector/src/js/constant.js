/**
 * constant.js
 */

/* string */
export const ALPHA_NUM = '[A-Z\\d]+';
export const AN_PLUS_B = 'AnPlusB';
export const COMBINATOR = 'Combinator';
export const IDENTIFIER = 'Identifier';
export const NOT_SUPPORTED_ERR = 'NotSupportedError';
export const NTH = 'Nth';
export const RAW = 'Raw';
export const SELECTOR = 'Selector';
export const SELECTOR_ATTR = 'AttributeSelector';
export const SELECTOR_CLASS = 'ClassSelector';
export const SELECTOR_ID = 'IdSelector';
export const SELECTOR_LIST = 'SelectorList';
export const SELECTOR_PSEUDO_CLASS = 'PseudoClassSelector';
export const SELECTOR_PSEUDO_ELEMENT = 'PseudoElementSelector';
export const SELECTOR_TYPE = 'TypeSelector';
export const STRING = 'String';
export const SYNTAX_ERR = 'SyntaxError';
export const U_FFFD = '\uFFFD';

/* numeric */
export const BIT_01 = 1;
export const BIT_02 = 2;
export const BIT_04 = 4;
export const BIT_08 = 8;
export const BIT_16 = 0x10;
export const BIT_32 = 0x20;
export const BIT_HYPHEN = 0x2D;
export const DUO = 2;
export const HEX = 16;
export const MAX_BIT_16 = 0xFFFF;
export const TYPE_FROM = 8;
export const TYPE_TO = -1;

/* Node */
export const ELEMENT_NODE = 1;
export const TEXT_NODE = 3;
export const DOCUMENT_NODE = 9;
export const DOCUMENT_FRAGMENT_NODE = 11;
export const DOCUMENT_POSITION_PRECEDING = 2;
export const DOCUMENT_POSITION_CONTAINS = 8;
export const DOCUMENT_POSITION_CONTAINED_BY = 0x10;

/* NodeFilter */
export const SHOW_ALL = 0xffffffff;
export const SHOW_DOCUMENT = 0x100;
export const SHOW_DOCUMENT_FRAGMENT = 0x400;
export const SHOW_ELEMENT = 1;

/* regexp */
export const REG_LOGICAL_PSEUDO = /^(?:(?:ha|i)s|not|where)$/;
export const REG_SHADOW_HOST = /^host(?:-context)?$/;
export const REG_SHADOW_MODE = /^(?:close|open)$/;
export const REG_SHADOW_PSEUDO = /^part|slotted$/;
