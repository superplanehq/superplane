"use strict";

/**
 * RegexParser
 * Parses a string input.
 *
 * @name RegexParser
 * @function
 * @param {String} input The string input that should be parsed as regular
 * expression.
 * @return {RegExp} The parsed regular expression.
 */
var RegexParser = module.exports = function (input) {
    // Validate input
    if (typeof input !== "string") {
        throw new Error("Invalid input. Input must be a string");
    }

    // Parse input
    var m = input.match(/(\/?)(.+)\1([a-z]*)/i);

    // If there's no match, throw an error
    if (!m) {
        throw new Error("Invalid regular expression format.");
    }

    // Filter valid flags: 'g', 'i', 'm', 's', 'u', and 'y'
    var validFlags = Array.from(new Set(m[3])).filter(function (flag) {
        return "gimsuy".includes(flag);
    }).join("");

    // Create the regular expression
    return new RegExp(m[2], validFlags);
};