# goinept

## Usage

```
./goinept -input encrypted.epub -key adobekey.der -output decrypted.epub
```

### Compression Level

`goinept` offers nine levels of compression, with each higher level trading speed for compression efficiency. By default, `goinept` uses level 5. To disable compression, you can use `--level 0`.

## Benchmarks

| Input file | File size  | ineptepub.py | goinept\* | Speedup |
| ---------- | ---------- | ------------ | --------- | ------- |
| tbd.epub   | 729 KB     | 0.81s        | 0.05s     | 16.2x   |
| sha.epub   | 2,459 KB   | 1.16s        | 0.08s     | 14.5x   |
| tlo.epub   | 4,113 KB   | 1.28s        | 0.06s     | 21.3x   |
| atk.epub   | 617,183 KB | 25.11s       | 3.87s     | 6.5x    |

\*`goinept` was run with the default compression level (5).

## Building CLI

```
go build -o goinept cmd/goinept/main.go
```

## Building WASM

```
GOOS=js GOARCH=wasm go build -o static/app.wasm lib/wasm/app.go
```

Run an HTTP server in the `static` directory to use the web interface.

## Notes

The `internal/zip` folder is a slightly modified version of the `archive/zip` library that allows access to the local file headers of zip files. It's current as of go `go1.16.4`.
