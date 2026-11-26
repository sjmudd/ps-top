// Package mapper turns a raw key, as collected, into the key a view shows. A
// view's MapKey is one of these mappers, or nil to show keys as collected.
//
// Each kind of mapper has its own file:
//
//   - filename.go: FileName maps a file name to the table or server file it
//     belongs to, e.g. /var/lib/mysql/shop/orders.ibd to shop.orders.
//   - prefix.go: TrimPrefix removes a fixed prefix, e.g. "stage/sql/" from
//     stage names.
//   - part.go: Part keeps one part of a key of several parts, e.g. the user
//     of a connection's key.
package mapper
