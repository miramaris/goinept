# goinept

## Usage

```
./goinept -input encrypted.epub -key adobekey.der -output decrypted.epub
```

## Benchmarks

| Input file | File size  | ineptepub.py | goinept | Speedup |
| ---------- | ---------- | ------------ | ------- | ------- |
| tbd.epub   | 729 KB     | 0.81s        | 0.07s   | 11.6x   |
| sha.epub   | 2,459 KB   | 1.16s        | 0.18s   | 6.4x    |
| tlo.epub   | 4,113 KB   | 1.28s        | 0.22s   | 5.8x    |
| atk.epub   | 617,183 KB | 25.11s       | 18.76s  | 1.3x    |

Note: Results for `ineptepub.py` do not include the time spent processing with `zipfix.py`, which needs to be run on certain ePUB files before they're able to be correctly decrypted. `goinept` is able to handle these files without any sort of preprocessing.

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
