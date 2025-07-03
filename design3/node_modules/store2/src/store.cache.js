/**
 * Copyright (c) 2013 ESHA Research
 * Dual licensed under the MIT and GPL licenses:
 *   http://www.opensource.org/licenses/mit-license.php
 *   http://www.gnu.org/licenses/gpl.html
 *
 * Allows use of a number as 'overwrite' param on set calls to give time in seconds after
 * which the value should not be retrievable again (an expiration time).
 *
 * Status: BETA - useful, needs testing
 */
;(function(_) {
    var prefix = 'exp@',
        suffix = ';',
        parse = _.parse,
        _get = _.get,
        _set = _.set;
    _.parse = function(s, fn) {
        if (s && s.indexOf(prefix) === 0) {
            s = s.substring(s.indexOf(suffix)+1);
        }
        return parse(s, fn);
    };
    _.expires = function(s) {
        if (s && s.indexOf(prefix) === 0) {
            return parseInt(s.substring(prefix.length, s.indexOf(suffix)), 10);
        }
        return false;
    };
    _.when = function(sec) {// if sec, return sec->date, else date->sec
        var now = Math.floor((new Date().getTime())/1000);
        return sec ? new Date((now+sec)*1000) : now;
    };
    _.cache = function(area, key) {
        var s = _get(area, key),
            sec = _.expires(s);
        if (sec && _.when() >= sec) {
            return area.removeItem(key);
        }
        return s;
    };
    _.get = function(area, key) {
        var s = _.cache(area, key);
        return s === undefined ? null : s;
    };
    _.set = function(area, key, string, sec) {
        try {
            if (sec) {
                string = prefix + (_.when()+sec) + suffix + string;
            }
            _set(area, key, string);
        } catch (e) {
            if (e.name === 'QUOTA_EXCEEDED_ERR' || e.name === 'NS_ERROR_DOM_QUOTA_REACHED') {
                var changed = false;
                for (var i=0,m=area.length; i<m; i++) {
                    if (_.cache(area, key) === undefined) {
                        changed = true;
                    }
                }
                if (changed) {
                    return _.set.apply(this, arguments);
                }
            }
            throw e;
        }
    };
})(window.store._, undefined);
