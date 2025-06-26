var mixin = require('./mixin.js');
var index = require('./index.js');

// Default Export

// Add all of the Underscore functions to the wrapper object.
var _ = mixin(index);
// Legacy Node.js API.
_._ = _;

module.exports = _;
