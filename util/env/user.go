package env

// MyUID is the Unix numeric user ID of the user running the
// application, or 0 on platforms that don't use Unix-style user
// accounts.
var MyUID uint32

// MyGID is the Unix numeric primary group ID of the user running the
// application, or 0 on platforms that don't use Unix-style user
// accounts.
var MyGID uint32
