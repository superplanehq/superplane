/*!
 * DOM Selector - A CSS selector engine.
 * @license MIT
 * @copyright asamuzaK (Kazz)
 * @see {@link https://github.com/asamuzaK/domSelector/blob/main/LICENSE}
 */

/* import */
import { Matcher } from './js/matcher.js';

/**
 * matches
 * @param {string} selector - CSS selector
 * @param {object} node - Element node
 * @param {object} [opt] - options
 * @param {boolean} [opt.warn] - console warn e.g. unsupported pseudo-class
 * @returns {boolean} - `true` if matched, `false` otherwise
 */
export const matches = (selector, node, opt) =>
  new Matcher(selector, node, opt).matches();

/**
 * closest
 * @param {string} selector - CSS selector
 * @param {object} node - Element node
 * @param {object} [opt] - options
 * @param {boolean} [opt.warn] - console warn e.g. unsupported pseudo-class
 * @returns {?object} - matched node
 */
export const closest = (selector, node, opt) =>
  new Matcher(selector, node, opt).closest();

/**
 * querySelector
 * @param {string} selector - CSS selector
 * @param {object} node - Document, DocumentFragment or Element node
 * @param {object} [opt] - options
 * @param {boolean} [opt.warn] - console warn e.g. unsupported pseudo-class
 * @returns {?object} - matched node
 */
export const querySelector = (selector, node, opt) =>
  new Matcher(selector, node, opt).querySelector();

/**
 * querySelectorAll
 * NOTE: returns Array, not NodeList
 * @param {string} selector - CSS selector
 * @param {object} node - Document, DocumentFragment or Element node
 * @param {object} [opt] - options
 * @param {boolean} [opt.warn] - console warn e.g. unsupported pseudo-class
 * @returns {Array.<object|undefined>} - array of matched nodes
 */
export const querySelectorAll = (selector, node, opt) =>
  new Matcher(selector, node, opt).querySelectorAll();
