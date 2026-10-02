# Changelog

## v0.1.1 (2026-10-02)

### Security

- **Network errors no longer contain the request URL.** A connection failure, timeout or cancelled context used to wrap Go's `*url.Error`, whose message includes the full request URL. For `Blockchain.Wallet.Generate` with a `Mnemonic`, that put the mnemonic, from the query string, into the error string. For `Blockchain.Wallet.DeriveAddress` on EGLD, it put the mnemonic from the path there too. Any ordinary error logging, such as `log.Println(err)` or an error tracker, then recorded the seed phrase. Every error now names only the HTTP method and the documented path template, for example `cryptures: GET /api/v1/blockchain/wallet/{chain}: dial tcp ...: connection refused`. The error still unwraps to a `*url.Error`, whose `URL` field is now that template, and to `net.Error`, so `errors.As`, `errors.Is(err, context.DeadlineExceeded)` and timeout checks keep working.
- `APIError.Message`, `Code` and `RequestID` now have CR, LF and other control characters replaced with spaces, and each is capped at 1024 characters. This stops a malicious or buggy upstream from injecting forged log lines. The unmodified response is still in `APIError.Body`.

### Changed

- **Retry behavior is now identical across the Go, JavaScript and Python SDKs.** `Blockchain.Wallet.Generate` is no longer retried automatically, because a retry returns a different freshly generated wallet. The README lists every operation that is not retried.

### Removed

- The unused internal helper `queryBuilder.strPtr`. This has no effect on the public API.
