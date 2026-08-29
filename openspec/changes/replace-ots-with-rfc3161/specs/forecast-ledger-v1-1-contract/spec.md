## REMOVED Requirements

### Requirement: Exact immutable v1.1.0 contract
**Reason**: Forecast Ledger v1.2.0 supersedes v1.1.0 and replaces its OTS timestamp object with RFC 3161.

**Migration**: None. This is a pre-adoption breaking cutover; v1.1.0 ledgers are unsupported input.

### Requirement: Exclusive schema-version acceptance
**Reason**: The old requirement exclusively admits v1.1.0, which is no longer the application contract.

**Migration**: None. Callers must author a new v1.2.0 ledger; the application provides no converter or dual-read mode.

### Requirement: Published v1.1.0 conformance corpus
**Reason**: Release conformance moves to the exact upstream v1.2.0 corpus and pins.

**Migration**: None. v1.1.0 fixtures remain upstream history and are not accepted by the application.

### Requirement: One schema identity across public surfaces
**Reason**: Every public surface must report the v1.2.0 identity instead of v1.1.0.

**Migration**: None. Consumers must use the new v1.2.0 metadata contract.
