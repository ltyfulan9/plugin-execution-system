# Release and Supply Chain Baseline

PES release artifacts must provide at least SHA-256 checksums. Enterprise release channels should add cosign signatures, container image signing, SBOM, and provenance attestation.

Required release gates:

1. `go test ./...`
2. `go test -race ./internal/service ./internal/worker ./internal/router`
3. `govulncheck ./...`
4. container image build
5. artifact checksum generation
6. SBOM generation
7. cosign signing for checksums and images
8. changelog and migration notes

The local scripts provide a zero-dependency baseline:

```bash
scripts/sbom.sh dist/sbom.txt
scripts/checksums.py dist > dist/checksums.txt
```
