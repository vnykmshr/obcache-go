module batch-processing-example

go 1.21

replace github.com/vnykmshr/obcache-go => ../..

require github.com/vnykmshr/obcache-go v0.0.0-00010101000000-000000000000

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/redis/go-redis/v9 v9.0.5 // indirect
)