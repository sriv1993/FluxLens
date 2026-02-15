# Contributing to FluxLens

Thanks for your interest in FluxLens. The project is in early active
development; contributions are welcome from issue triage through code
contributions.

## Code of Conduct

This project follows the
[Contributor Covenant v2.1](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).
See [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md). Participants are
expected to abide by it.

## Getting set up

```bash
git clone https://github.com/sriharshav1/fluxlens.git
cd fluxlens
make tidy
make test
```

To run the local development stack:

```bash
make dev        # start kafka, postgres, redis, mock LLM, prometheus, grafana
make build      # build the FluxLens binaries
./bin/fluxlens-synth-source --kafka localhost:9092 --rate 100
./bin/fluxlens-curator --kafka localhost:9092 --strategy 4 --diversity 80
```

## Branching and commits

- `main` is always shippable.
- Feature branches use the form `feat/<short-name>` or
  `fix/<short-name>` or `docs/<short-name>`.
- All commits must be signed off via [DCO](https://developercertificate.org/)
  (`git commit -s`).
- Commit messages follow the
  [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
  format: `type(scope): summary`.

## Pull request process

1. Open an issue first for non-trivial changes so we can discuss design.
2. Open a draft PR early if you want feedback during development.
3. Every PR must:
   - Pass `make test`
   - Pass `make lint`
   - Maintain or improve code coverage
   - Update documentation when API or behavior changes
   - Include an ADR under `docs/adr/` for non-trivial architectural changes
4. PRs are reviewed by maintainers. A second reviewer joins once
   a steering committee is formed (planned Phase 2).

## Architecture Decision Records (ADRs)

Non-trivial architectural changes require an ADR. Use the template at
`docs/adr/template.md`. Number ADRs sequentially.

## Testing expectations

- Every new package includes unit tests with table-driven cases.
- Integration tests live under `internal/<package>/integration_test.go`
  and run in CI against the local docker-compose stack.
- Aim for at least 70% line coverage at PR time (target 80% by Phase 2).

## Security

If you discover a security vulnerability, please do not file a public
issue. See [SECURITY.md](./SECURITY.md) for the responsible disclosure
process.

## Licensing

By contributing, you agree that your contributions are licensed under
the Apache License 2.0 (see [LICENSE](./LICENSE)) and that you have
the right to submit them under those terms.
