package auditlog

// Store is the append-only hash-chained audit log used by the gateway,
// orchestrator, and audit-writer. Implementations must be safe for
// concurrent Append from a single writer process; Postgres deployments
// should use one writer service per tenant chain.
type Store interface {
	Append(kind string, payload any) (Record, error)
	Verify() (bool, error)
	Len() int
	HeadHash() string
	Snapshot() []Record
}

// Ensure Chain implements Store.
var _ Store = (*Chain)(nil)
