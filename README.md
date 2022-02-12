# qpprof

qpprof complements the [pprof](https://github.com/google/pprof) tool.

## Commands

Use `qpprof command --help` to get more information.

### Flat aggregation

Alternative flat aggregations allow you to get the `top` with some of the
nodes being folded to their callers.

For instance, take this `copyBytes()` function:

```go
func copyBytes(b []byte) []byte {
  dst := make([]byte, len(b))
  copy(dst, b)
  return dst
}
```

If you benchmark it and use `pprof top`, then you'll see this:

```
(pprof) top 5
37.56% runtime.mallocgc
12.16% runtime.memclrNoHeapPointers
 9.35% runtime.memmove
 8.47% runtime.scanobject
 6.42% runtime.scanblock
```

With `flat-with-runtime` you'll get something that you would expect:

```bash
$ qpprof flat-with-runtime cpu.out
6.25s example.copyBytes
 40ms example.BenchmarkCopyBytes
 20ms testing.(*B).runN
```

There is also `flat-with-stdlib` that folds all standard library functions,
not just the `runtime` package.

### Enriching the profile

Given a CPU profile `X` and executable `E` that was used to collect it, we can
generate a new CPU profile `Y` that contains even more useful information.

What we can add to the profile:

* Explicit bound checks timings (displayed as `runtime.boundcheck`)
* Explicit nil checks timings (displayed as `runtime.nilcheck`)

```bash
$ qpprof enrich -o=cpu2.out -exe=prog cpu.out
runtime.boundcheck: 7 samples (300ms)
runtime.nilcheck: 22 samples (300ms)
```

Now let's open `cpu2.out` with pprof:

```
(pprof) top boundcheck|nilcheck
Active filters:
   focus=boundcheck|nilcheck
Showing nodes accounting for 0.60s, 1.10% of 54.68s total
Showing top 10 nodes out of 132
      flat  flat%   sum%        cum   cum%
     0.30s  0.55%  0.55%      0.30s  0.55%  runtime.boundcheck (inline)
     0.30s  0.55%  1.10%      0.30s  0.55%  runtime.nilcheck (inline)
```

The implicit bound checks are now explicit, visible in the profile.
