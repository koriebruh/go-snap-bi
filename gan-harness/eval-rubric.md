# GAN Eval Rubric — go-snap-bi Slice 1 (Signing core)

Score each run 0-10 per dimension. Passing bar: all dimensions >= 8, no
dimension below 6. Evaluator runs `go build ./...` and `go test ./... -v
-race` first — a build failure or race is an automatic 0 on Correctness
regardless of other dimensions.

## Correctness (weight: highest)

- `go build ./...` succeeds, `go vet ./...` clean.
- `go test ./... -race` passes, including every table case in
  `gan-harness/spec.md`'s "Required tests" section.
- Symmetric and asymmetric `stringToSign` formulas match the spec exactly,
  including the presence/absence of the `AccessToken` segment.
- Empty-body case hashes an empty byte slice, does not special-case into an
  error or a literal empty string hash mismatch.
- `SignAsymmetric`/`VerifyAsymmetric` genuinely round-trip with a real RSA
  keypair generated in the test, not stubbed/faked crypto.

## Security

- `VerifySymmetric` uses `hmac.Equal` or equivalent constant-time compare —
  a plain `==` string comparison on a computed HMAC is a finding, not a pass.
- No secret material (client secret, private key bytes) logged, wrapped into
  error strings, or otherwise leaked.
- `ParseRSAPrivateKeyPEM` rejects non-RSA key types and malformed PEM with a
  typed error, does not panic on malformed input.

## API shape fidelity to the design doc

- `SignAsymmetric` signature takes `crypto.Signer`, not `*rsa.PrivateKey` or
  raw PEM bytes — this is a named design decision (HSM/KMS pluggability),
  not a style preference. Any deviation is an automatic finding.
- Function names, parameter order, and return types match
  `gan-harness/spec.md` exactly (downstream slices depend on this shape).
- No extra public API surface beyond what's listed (no premature exported
  helpers, no config structs not asked for) — YAGNI applies to the library's
  own surface.

## Test quality

- Table-driven, stdlib `testing` only — a test file importing testify,
  gomock, or any third-party test helper is a finding.
- Tests fail meaningfully if the implementation is wrong (mutate one
  character of an expected `stringToSign` mentally and confirm the test
  would catch it) — not tests that only check "no error returned."

## Idiomatic Go / simplicity

- No unrequested abstractions (no interface with a single implementation
  invented for this slice, no factory functions).
- Errors wrapped with `%w` where a caller might reasonably want
  `errors.Is`/`errors.As` (e.g. PEM parse failures).
- No premature generics, no reflection, no unsafe.

## Scoring notes

- A generator output that skips a required test case scores <= 5 on Test
  quality regardless of implementation correctness.
- A generator output that adds Slice 2/3/4 functionality early scores <= 5
  on API shape fidelity — scope creep is a defect here, not a bonus.
