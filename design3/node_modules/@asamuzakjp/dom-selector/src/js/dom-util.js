/**
 * dom-util.js
 */

/* import */
import bidiFactory from 'bidi-js';

/* constants */
import {
  DOCUMENT_FRAGMENT_NODE, DOCUMENT_NODE, DOCUMENT_POSITION_CONTAINS,
  DOCUMENT_POSITION_CONTAINED_BY, DOCUMENT_POSITION_PRECEDING, ELEMENT_NODE,
  REG_SHADOW_MODE, SYNTAX_ERR, TEXT_NODE
} from './constant.js';

/**
 * is in shadow tree
 * @param {object} node - node
 * @returns {boolean} - result;
 */
export const isInShadowTree = (node = {}) => {
  let bool;
  if (node.nodeType === ELEMENT_NODE ||
      node.nodeType === DOCUMENT_FRAGMENT_NODE) {
    let refNode = node;
    while (refNode) {
      const { host, mode, nodeType, parentNode } = refNode;
      if (host && mode && nodeType === DOCUMENT_FRAGMENT_NODE &&
          REG_SHADOW_MODE.test(mode)) {
        bool = true;
        break;
      }
      refNode = parentNode;
    }
  }
  return !!bool;
};

/**
 * get slotted text content
 * @param {object} node - Element node
 * @returns {?string} - text content
 */
export const getSlottedTextContent = (node = {}) => {
  let res;
  if (node.localName === 'slot' && isInShadowTree(node)) {
    const nodes = node.assignedNodes();
    if (nodes.length) {
      for (const item of nodes) {
        res = item.textContent.trim();
        if (res) {
          break;
        }
      }
    } else {
      res = node.textContent.trim();
    }
  }
  return res ?? null;
};

/**
 * get directionality of node
 * @see https://html.spec.whatwg.org/multipage/dom.html#the-dir-attribute
 * @param {object} node - Element node
 * @returns {?string} - 'ltr' / 'rtl'
 */
export const getDirectionality = (node = {}) => {
  let res;
  if (node.nodeType === ELEMENT_NODE) {
    const { dir: nodeDir, localName, parentNode } = node;
    const { getEmbeddingLevels } = bidiFactory();
    const regDir = /^(?:ltr|rtl)$/;
    if (regDir.test(nodeDir)) {
      res = nodeDir;
    } else if (nodeDir === 'auto') {
      let text;
      switch (localName) {
        case 'input': {
          if (!node.type || /^(?:(?:butto|hidde)n|(?:emai|te|ur)l|(?:rese|submi|tex)t|password|search)$/.test(node.type)) {
            text = node.value;
          }
          break;
        }
        case 'slot': {
          text = getSlottedTextContent(node);
          break;
        }
        case 'textarea': {
          text = node.value;
          break;
        }
        default: {
          const items = [].slice.call(node.childNodes);
          for (const item of items) {
            const {
              dir: itemDir, localName: itemLocalName, nodeType: itemNodeType,
              textContent: itemTextContent
            } = item;
            if (itemNodeType === TEXT_NODE) {
              text = itemTextContent.trim();
            } else if (itemNodeType === ELEMENT_NODE) {
              if (!/^(?:bdi|s(?:cript|tyle)|textarea)$/.test(itemLocalName) &&
                  (!itemDir || !regDir.test(itemDir))) {
                if (itemLocalName === 'slot') {
                  text = getSlottedTextContent(item);
                } else {
                  text = itemTextContent.trim();
                }
              }
            }
            if (text) {
              break;
            }
          }
        }
      }
      if (text) {
        const { paragraphs: [{ level }] } = getEmbeddingLevels(text);
        if (level % 2 === 1) {
          res = 'rtl';
        } else {
          res = 'ltr';
        }
      }
      if (!res) {
        if (parentNode) {
          const { nodeType: parentNodeType } = parentNode;
          if (parentNodeType === ELEMENT_NODE) {
            res = getDirectionality(parentNode);
          } else if (parentNodeType === DOCUMENT_NODE ||
                     parentNodeType === DOCUMENT_FRAGMENT_NODE) {
            res = 'ltr';
          }
        } else {
          res = 'ltr';
        }
      }
    } else if (localName === 'bdi') {
      const text = node.textContent.trim();
      if (text) {
        const { paragraphs: [{ level }] } = getEmbeddingLevels(text);
        if (level % 2 === 1) {
          res = 'rtl';
        } else {
          res = 'ltr';
        }
      }
      if (!(res || parentNode)) {
        res = 'ltr';
      }
    } else if (localName === 'input' && node.type === 'tel') {
      res = 'ltr';
    } else if (parentNode) {
      if (localName === 'slot') {
        const text = getSlottedTextContent(node);
        if (text) {
          const { paragraphs: [{ level }] } = getEmbeddingLevels(text);
          if (level % 2 === 1) {
            res = 'rtl';
          } else {
            res = 'ltr';
          }
        }
      }
      if (!res) {
        const { nodeType: parentNodeType } = parentNode;
        if (parentNodeType === ELEMENT_NODE) {
          res = getDirectionality(parentNode);
        } else if (parentNodeType === DOCUMENT_NODE ||
                   parentNodeType === DOCUMENT_FRAGMENT_NODE) {
          res = 'ltr';
        }
      }
    } else {
      res = 'ltr';
    }
  }
  return res ?? null;
};

/**
 * is content editable
 * NOTE: not implemented in jsdom https://github.com/jsdom/jsdom/issues/1670
 * @param {object} node - Element node
 * @returns {boolean} - result
 */
export const isContentEditable = (node = {}) => {
  let res;
  if (node.nodeType === ELEMENT_NODE) {
    if (typeof node.isContentEditable === 'boolean') {
      res = node.isContentEditable;
    } else if (node.ownerDocument.designMode === 'on') {
      res = true;
    } else if (node.hasAttribute('contenteditable')) {
      const attr = node.getAttribute('contenteditable');
      if (attr === '' || /^(?:plaintext-only|true)$/.test(attr)) {
        res = true;
      } else if (attr === 'inherit') {
        let parent = node.parentNode;
        while (parent) {
          if (isContentEditable(parent)) {
            res = true;
            break;
          }
          parent = parent.parentNode;
        }
      }
    }
  }
  return !!res;
};

/**
 * is namespace declared
 * @param {string} ns - namespace
 * @param {object} node - Element node
 * @returns {boolean} - result
 */
export const isNamespaceDeclared = (ns = '', node = {}) => {
  let res;
  if (ns && typeof ns === 'string' && node.nodeType === ELEMENT_NODE) {
    const attr = `xmlns:${ns}`;
    const root = node.ownerDocument.documentElement;
    let parent = node;
    while (parent) {
      if (typeof parent.hasAttribute === 'function' &&
          parent.hasAttribute(attr)) {
        res = true;
        break;
      } else if (parent === root) {
        break;
      }
      parent = parent.parentNode;
    }
  }
  return !!res;
};

/**
 * is inclusive - nodeA and nodeB are in inclusive relation
 * @param {object} nodeA - Element node
 * @param {object} nodeB - Element node
 * @returns {boolean} - result
 */
export const isInclusive = (nodeA = {}, nodeB = {}) => {
  let res;
  if (nodeA.nodeType === ELEMENT_NODE && nodeB.nodeType === ELEMENT_NODE) {
    const posBit = nodeB.compareDocumentPosition(nodeA);
    res = posBit & DOCUMENT_POSITION_CONTAINS ||
          posBit & DOCUMENT_POSITION_CONTAINED_BY;
  }
  return !!res;
};

/**
 * is preceding - nodeA precedes and/or contains nodeB
 * @param {object} nodeA - Element node
 * @param {object} nodeB - Element node
 * @returns {boolean} - result
 */
export const isPreceding = (nodeA = {}, nodeB = {}) => {
  let res;
  if (nodeA.nodeType === ELEMENT_NODE && nodeB.nodeType === ELEMENT_NODE) {
    const posBit = nodeB.compareDocumentPosition(nodeA);
    res = posBit & DOCUMENT_POSITION_PRECEDING ||
          posBit & DOCUMENT_POSITION_CONTAINS;
  }
  return !!res;
};

/**
 * selector to node properties - e.g. ns|E -> { prefix: ns, tagName: E }
 * @param {string} selector - type selector
 * @param {object} [node] - Element node
 * @returns {object} - node properties
 */
export const selectorToNodeProps = (selector, node) => {
  let prefix;
  let tagName;
  if (selector && typeof selector === 'string') {
    if (selector.indexOf('|') > -1) {
      [prefix, tagName] = selector.split('|');
    } else {
      prefix = '*';
      tagName = selector;
    }
  } else {
    throw new DOMException(`Invalid selector ${selector}`, SYNTAX_ERR);
  }
  return {
    prefix,
    tagName
  };
};
