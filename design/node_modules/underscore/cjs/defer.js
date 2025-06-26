var underscore = require('./underscore.js');
var partial = require('./partial.js');
var delay = require('./delay.js');

// Defers a function, scheduling it to run after the current call stack has
// cleared.
var defer = partial(delay, underscore, 1);

module.exports = defer;
