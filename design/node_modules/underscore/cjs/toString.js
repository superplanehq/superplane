var value = require('./value.js');

// Provide an unwrapping proxy for automatic string coercion in engine
// operations such as JSON stringification.
function toString(wrapper) {
  return String(value(wrapper));
}

module.exports = toString;
