# rate limit buffer

Not a super profiled data structure, but seems to be ok for an http
router. I dropped previous concurrency tests, because these were
miss leading (measuring goroutine spawn in the benchmark loop)

    # 8e146dfb7ae73ae1d80c25108d64d5431c1efbe4
    % go test -bench=. -benchmem -cpu 1,2,4,8
    goos: darwin
    goarch: amd64
    pkg: github.com/szuecs/rate-limit-buffer
    BenchmarkAllow                          1000000              3192 ns/op             242 B/op          4 allocs/op
    BenchmarkAllow-2                        1000000              1715 ns/op             243 B/op          4 allocs/op
    BenchmarkAllow-4                        1000000              1661 ns/op             243 B/op          4 allocs/op
    BenchmarkAllow-8                        1000000              1430 ns/op             243 B/op          4 allocs/op


    # 4879455378dd0207c7a41ec4256023562d4529cb
    % go test -bench=. -benchmem -cpu 1,2,4,8
    goos: darwin
    goarch: amd64
    pkg: github.com/szuecs/rate-limit-buffer
    BenchmarkAllow                          1000000              2501 ns/op             243 B/op          4 allocs/op
    BenchmarkAllow-2                        1000000              1468 ns/op             243 B/op          4 allocs/op
    BenchmarkAllow-4                        1000000              1708 ns/op             243 B/op          4 allocs/op
    BenchmarkAllow-8                        1000000              1405 ns/op             242 B/op          4 allocs/op

    # Preprovisioned data set: 67bb879964b58763a7ca3d429f10fe5f7bbe644c
    % go test -bench=BenchmarkAllowBaseData1 -benchmem -cpu 1,2,4,8
    goos: darwin
    goarch: amd64
    pkg: github.com/szuecs/rate-limit-buffer
    BenchmarkAllowBaseData1                 5000000               315 ns/op              14 B/op          1 allocs/op
    BenchmarkAllowBaseData1-2               5000000               296 ns/op              14 B/op          1 allocs/op
    BenchmarkAllowBaseData1-4               5000000               290 ns/op              14 B/op          1 allocs/op
    BenchmarkAllowBaseData1-8               5000000               288 ns/op              14 B/op          1 allocs/op
    BenchmarkAllowBaseData10                5000000               319 ns/op              14 B/op          1 allocs/op
    BenchmarkAllowBaseData10-2              5000000               287 ns/op              14 B/op          1 allocs/op
    BenchmarkAllowBaseData10-4              5000000               283 ns/op              14 B/op          1 allocs/op
    BenchmarkAllowBaseData10-8              5000000               287 ns/op              14 B/op          1 allocs/op
    BenchmarkAllowBaseData100               5000000               311 ns/op              15 B/op          1 allocs/op
    BenchmarkAllowBaseData100-2             5000000               305 ns/op              15 B/op          1 allocs/op
    BenchmarkAllowBaseData100-4             5000000               299 ns/op              15 B/op          1 allocs/op
    BenchmarkAllowBaseData100-8             5000000               303 ns/op              15 B/op          1 allocs/op
    BenchmarkAllowBaseData1000              3000000               439 ns/op              21 B/op          2 allocs/op
    BenchmarkAllowBaseData1000-2            3000000               404 ns/op              21 B/op          2 allocs/op
    BenchmarkAllowBaseData1000-4            3000000               396 ns/op              21 B/op          2 allocs/op
    BenchmarkAllowBaseData1000-8            3000000               403 ns/op              21 B/op          2 allocs/op
    PASS
    ok      github.com/szuecs/rate-limit-buffer     53.970s

I am unsure how to measure with benchmark correctly the concurrent
behavior of this data structure. It's clear that high contention will
result in less throughput. 1,10,100,1000 are the number of goroutines
working concurrently on this data. 300µs/op is ok for 1000 goroutines
reading and writing the dataset, compared to do a proxy http call it
is much faster and will almost never happen at this concurrency level.

    # Concurrent/Contention tests: efcb5016a8a33a8f271ae9a5a527a0433ed727e4
    % go test -bench=BenchmarkAllowConcurrent1 -benchmem -cpu 1,2,4,8
    goos: darwin
    goarch: amd64
    pkg: github.com/szuecs/rate-limit-buffer
    BenchmarkAllowConcurrent1               5000000               364 ns/op              15 B/op          1 allocs/op
    BenchmarkAllowConcurrent1-2             5000000               319 ns/op              15 B/op          1 allocs/op
    BenchmarkAllowConcurrent1-4             5000000               313 ns/op              15 B/op          1 allocs/op
    BenchmarkAllowConcurrent1-8             5000000               332 ns/op              15 B/op          1 allocs/op
    BenchmarkAllowConcurrent10               500000              3383 ns/op             159 B/op         19 allocs/op
    BenchmarkAllowConcurrent10-2             500000              2204 ns/op             159 B/op         19 allocs/op
    BenchmarkAllowConcurrent10-4             500000              3325 ns/op             159 B/op         19 allocs/op
    BenchmarkAllowConcurrent10-8             500000              2582 ns/op             159 B/op         19 allocs/op
    BenchmarkAllowConcurrent100               50000             34248 ns/op            1593 B/op        199 allocs/op
    BenchmarkAllowConcurrent100-2            100000             21281 ns/op            1592 B/op        199 allocs/op
    BenchmarkAllowConcurrent100-4             50000             36870 ns/op            1593 B/op        199 allocs/op
    BenchmarkAllowConcurrent100-8             50000             35154 ns/op            1593 B/op        199 allocs/op
    BenchmarkAllowConcurrent1000               5000            319270 ns/op           15928 B/op       1990 allocs/op
    BenchmarkAllowConcurrent1000-2            10000            230893 ns/op           15936 B/op       1990 allocs/op
    BenchmarkAllowConcurrent1000-4             5000            344456 ns/op           15946 B/op       1990 allocs/op
    BenchmarkAllowConcurrent1000-8             3000            333466 ns/op           15996 B/op       1990 allocs/op
    PASS
    ok      github.com/szuecs/rate-limit-buffer     42.174s
