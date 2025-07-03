export declare function EBUSY(message: string): Error & {
    code: string;
};
export declare function ENOSYS(message: string, reason: string): Error & {
    code: string;
};
export declare function EINVAL(reason: string): Error & {
    code: string;
};
export declare function EBADF(reason: string): Error & {
    code: string;
};
export declare function ENOENT(reason: string): Error & {
    code: string;
};
export declare function ENOTDIR(reason: string): Error & {
    code: string;
};
export declare function EISDIR(reason: string): Error & {
    code: string;
};
export declare function EEXIST(reason: string): Error & {
    code: string;
};
export declare function EROFS(reason: string): Error & {
    code: string;
};
export declare function ENOTEMPTY(reason: string): Error & {
    code: string;
};
export declare function EOPNOTSUPP(reason: string): Error & {
    code: string;
};
export declare function ERR_DIR_CLOSED(): Error & {
    code: string;
};
export declare class LibzipError extends Error {
    code: string;
    constructor(message: string, code: string);
}
