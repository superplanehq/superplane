// Internal helper to generate a callback that will append
// its first argument to the closed-over `array`.
function pusher(array) {
  return function(arg) {
    array.push(arg);
  };
}

module.exports = pusher;
