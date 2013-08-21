// Package cas implements a Content-Addressed Store.
//
// Data stored in it will be cryptographically hashed into a Key, that
// can later be used to fetch the data. Storing the same data again
// will result in the same Key, and not take extra space.
package cas
