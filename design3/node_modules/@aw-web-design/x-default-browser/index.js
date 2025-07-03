var detect;
var os = require('os');

if (os.platform() == 'win32') {
    if (os.release().indexOf('10.') == 0)
        detect = require('./src/detect-windows10');
    else
        detect = require('./src/detect-windows');
} else if (os.platform() == 'darwin') {
    detect = require('./src/detect-mac');
} else if (os.platform() == 'linux' || os.platform() == 'freebsd') {
    detect = require('./src/detect-linux');
} else {
    detect = require('./src/detect-stub');
}

module.exports = detect;
