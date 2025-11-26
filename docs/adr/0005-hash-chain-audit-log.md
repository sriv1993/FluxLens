# ADR 0005 — Append-only hash-chained audit log

- **Status:** Accepted
- **Date:** 2026-05-16
- **Author:** Sri Harsha Vanga

## Context

FluxLens deployments serve sectors (clean-energy manufacturing,
critical retail and supply-chain operations, federally funded
research) where audit-trail integrity is operationally and
sometimes regulatorily required. Operators must be able to detect
tampering with historical records, both for after-action review and
for regulatory or warranty-claim defensibility.

Options considered for the audit-log architecture:

1. **Plain append-only log** (e.g., write-only Postgres table).
2. **Append-only hash-chained log** (each record contains the hash
   of its predecessor).
3. **Blockchain-style log** (distributed ledger with consensus).
4. **Signed batches** (periodic signing of batches of records by an
   external authority).

## Decision

FluxLens uses an **append-only hash-chained log** with optional
WORM (Write-Once-Read-Many) mirroring to S3 Object Lock in
compliance mode for operators requiring it.

## Rationale

A plain append-only log (option 1) does not provide tamper
evidence: an attacker with write access to the underlying storage
can rewrite history undetectably.

A blockchain-style log (option 3) provides strong tamper evidence
but introduces consensus and key-management complexity disproportionate
to the operational use case. FluxLens is single-operator (no
multi-party trust assumptions); a blockchain would impose
unnecessary cost.

Signed batches (option 4) require external signing authority and
introduce batching latency that is undesirable for real-time audit.

The hash-chained log (option 2) provides:

- Tamper evidence: any modification to a historical record breaks
  the chain at and after the modified record, detectable in O(n)
  by recomputing hashes.
- Simplicity: a single writer service; no consensus, no
  cross-organization key management.
- Real-time append: no batching latency.
- Auditor independence: the chain verifier process runs
  independent of the writer, detecting tampering even if the
  writer itself is compromised.

WORM mirroring (S3 Object Lock in compliance mode) provides the
additional tamper-resistance property that the underlying storage
cannot be modified, defending against insider threats with cluster
admin privileges.

## Consequences

- The audit log is append-only at the application level; deletion
  of historical records is not supported.
- Storage grows monotonically. Operators implement retention via
  partition-drop at the bottom of the chain (the chain remains
  intact above the dropped partition; the dropped partition is
  preserved in WORM storage for the required retention period).
- The chain verifier must run on a separate process/node from the
  writer to defend against writer compromise.
- Chain verification is O(n); for very long chains (>10M records)
  operators may prefer incremental verification or sub-chain
  verification.

## Alternatives considered

- **Plain append-only.** Rejected; insufficient tamper evidence.
- **Blockchain-style.** Rejected; complexity disproportionate to
  use case.
- **Signed batches.** Rejected; batching latency and external
  signer dependency.

## References

- NIST SP 800-92 (Guide to Computer Security Log Management)
- NIST SP 800-53 Rev. 5 AU-9 (Protection of Audit Information),
  AU-10 (Non-Repudiation)
- AWS S3 Object Lock Compliance Mode documentation
