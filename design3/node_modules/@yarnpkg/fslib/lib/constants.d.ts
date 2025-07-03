export declare const S_IFMT = 61440;
export declare const S_IFDIR = 16384;
export declare const S_IFREG = 32768;
export declare const S_IFLNK = 40960;
/**
 * Unix timestamp for `1984-06-22T21:50:00.000Z`
 *
 * It needs to be after 1980-01-01 because that's what Zip supports, and it
 * needs to have a slight offset to account for different timezones (because
 * zip assumes that all times are local to whoever writes the file, which is
 * really silly).
 */
export declare const SAFE_TIME = 456789000;
