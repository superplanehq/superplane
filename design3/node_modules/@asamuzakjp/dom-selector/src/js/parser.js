/**
 * parser.js
 */

/* import */
import { findAll, parse, toPlainObject, walk } from 'css-tree';

/* constants */
import {
  DUO, HEX, MAX_BIT_16, BIT_HYPHEN, REG_LOGICAL_PSEUDO, REG_SHADOW_PSEUDO,
  SELECTOR, SELECTOR_PSEUDO_CLASS, SELECTOR_PSEUDO_ELEMENT, SYNTAX_ERR,
  TYPE_FROM, TYPE_TO, U_FFFD
} from './constant.js';

/**
 * unescape selector
 * @param {string} selector - CSS selector
 * @returns {?string} - unescaped selector
 */
export const unescapeSelector = (selector = '') => {
  if (typeof selector === 'string' && selector.indexOf('\\', 0) >= 0) {
    const arr = selector.split('\\');
    const l = arr.length;
    for (let i = 1; i < l; i++) {
      let item = arr[i];
      if (item === '' && i === l - 1) {
        item = U_FFFD;
      } else {
        const hexExists = /^([\da-f]{1,6}\s?)/i.exec(item);
        if (hexExists) {
          const [, hex] = hexExists;
          let str;
          try {
            const low = parseInt('D800', HEX);
            const high = parseInt('DFFF', HEX);
            const deci = parseInt(hex, HEX);
            if (deci === 0 || (deci >= low && deci <= high)) {
              str = U_FFFD;
            } else {
              str = String.fromCodePoint(deci);
            }
          } catch (e) {
            str = U_FFFD;
          }
          let postStr = '';
          if (item.length > hex.length) {
            postStr = item.substring(hex.length);
          }
          item = `${str}${postStr}`;
        // whitespace
        } else if (/^[\n\r\f]/.test(item)) {
          item = '\\' + item;
        }
      }
      arr[i] = item;
    }
    selector = arr.join('');
  }
  return selector;
};

/**
 * preprocess
 * @see https://drafts.csswg.org/css-syntax-3/#input-preprocessing
 * @param {...*} args - arguments
 * @returns {string} - filtered selector string
 */
export const preprocess = (...args) => {
  if (!args.length) {
    throw new TypeError('1 argument required, but only 0 present.');
  }
  let [selector] = args;
  if (typeof selector === 'string') {
    let index = 0;
    while (index >= 0) {
      index = selector.indexOf('#', index);
      if (index < 0) {
        break;
      }
      const preHash = selector.substring(0, index + 1);
      let postHash = selector.substring(index + 1);
      const codePoint = postHash.codePointAt(0);
      // @see https://drafts.csswg.org/selectors/#id-selectors
      // @see https://drafts.csswg.org/css-syntax-3/#ident-token-diagram
      if (codePoint === BIT_HYPHEN) {
        if (/^\d$/.test(postHash.substring(1, 2))) {
          throw new DOMException(`Invalid selector ${selector}`, SYNTAX_ERR);
        }
      // escape char above 0xFFFF
      } else if (codePoint > MAX_BIT_16) {
        const str = `\\${codePoint.toString(HEX)} `;
        if (postHash.length === DUO) {
          postHash = str;
        } else {
          postHash = `${str}${postHash.substring(DUO)}`;
        }
      }
      selector = `${preHash}${postHash}`;
      index++;
    }
    selector = selector.replace(/\f|\r\n?/g, '\n')
      .replace(/[\0\uD800-\uDFFF]|\\$/g, U_FFFD);
  } else if (selector === undefined || selector === null) {
    selector = Object.prototype.toString.call(selector)
      .slice(TYPE_FROM, TYPE_TO).toLowerCase();
  } else if (Array.isArray(selector)) {
    selector = selector.join(',');
  } else if (Object.prototype.hasOwnProperty.call(selector, 'toString')) {
    selector = selector.toString();
  } else {
    throw new DOMException(`Invalid selector ${selector}`, SYNTAX_ERR);
  }
  return selector;
};

/**
 * create AST from CSS selector
 * @param {string} selector - CSS selector
 * @returns {object} - AST
 */
export const parseSelector = selector => {
  selector = preprocess(selector);
  // invalid selectors
  if (/^$|^\s*>|,\s*$/.test(selector)) {
    throw new DOMException(`Invalid selector ${selector}`, SYNTAX_ERR);
  }
  let res;
  try {
    const ast = parse(selector, {
      context: 'selectorList',
      parseCustomProperty: true
    });
    res = toPlainObject(ast);
  } catch (e) {
    // workaround for https://github.com/csstree/csstree/issues/265
    // NOTE: still throws on `:lang("")`;
    const regLang = /(:lang\(\s*("[A-Za-z\d\-*]+")\s*\))/;
    if (e.message === 'Identifier is expected' && regLang.test(selector)) {
      const [, lang, range] = regLang.exec(selector);
      const escapedRange =
        range.replaceAll('*', '\\*').replace(/^"/, '').replace(/"$/, '');
      const escapedLang = lang.replace(range, escapedRange);
      res = parseSelector(selector.replace(lang, escapedLang));
    } else if (e.message === '"]" is expected' && !selector.endsWith(']')) {
      res = parseSelector(`${selector}]`);
    } else if (e.message === '")" is expected' && !selector.endsWith(')')) {
      res = parseSelector(`${selector})`);
    } else {
      throw new DOMException(e.message, SYNTAX_ERR);
    }
  }
  return res;
};

/**
 * walk AST
 * @param {object} ast - AST
 * @returns {Array.<object|undefined>} - collection of AST branches
 */
export const walkAST = (ast = {}) => {
  const branches = new Set();
  let hasPseudoFunc;
  const opt = {
    enter: node => {
      if (node.type === SELECTOR) {
        branches.add(node.children);
      } else if ((node.type === SELECTOR_PSEUDO_CLASS &&
                  REG_LOGICAL_PSEUDO.test(node.name)) ||
                 (node.type === SELECTOR_PSEUDO_ELEMENT &&
                  REG_SHADOW_PSEUDO.test(node.name))) {
        hasPseudoFunc = true;
      }
    }
  };
  walk(ast, opt);
  if (hasPseudoFunc) {
    findAll(ast, (node, item, list) => {
      if (list) {
        if (node.type === SELECTOR_PSEUDO_CLASS &&
            REG_LOGICAL_PSEUDO.test(node.name)) {
          const itemList = list.filter(i => {
            const { name, type } = i;
            const res =
              type === SELECTOR_PSEUDO_CLASS && REG_LOGICAL_PSEUDO.test(name);
            return res;
          });
          for (const { children } of itemList) {
            // SelectorList
            for (const { children: grandChildren } of children) {
              // Selector
              for (const { children: greatGrandChildren } of grandChildren) {
                if (branches.has(greatGrandChildren)) {
                  branches.delete(greatGrandChildren);
                }
              }
            }
          }
        } else if (node.type === SELECTOR_PSEUDO_ELEMENT &&
                   REG_SHADOW_PSEUDO.test(node.name)) {
          const itemList = list.filter(i => {
            const { name, type } = i;
            const res =
              type === SELECTOR_PSEUDO_ELEMENT && REG_SHADOW_PSEUDO.test(name);
            return res;
          });
          for (const { children } of itemList) {
            // Selector
            for (const { children: grandChildren } of children) {
              if (branches.has(grandChildren)) {
                branches.delete(grandChildren);
              }
            }
          }
        }
      }
    });
  }
  return [...branches];
};

/* export */
export { generate as generateCSS } from 'css-tree';
