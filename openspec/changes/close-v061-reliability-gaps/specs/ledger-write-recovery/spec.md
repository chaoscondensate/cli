## Purpose

Ensures an interrupted single-ledger replacement is completed or refused safely before a later mutation proceeds.

## ADDED Requirements

### Requirement: Recover a valid ledger journal under the writer lock
Every single-ledger mutation SHALL acquire the ledger writer lock before inspecting an unfinished replacement journal. If the journal, current ledger, synchronized sibling temporary file, and prospective ledger validation prove one unambiguous recovery action, the application SHALL complete that action, durably remove the journal, and continue the requested mutation against the recovered ledger while retaining the same writer lock.

#### Scenario: Interruption before replacement
- **WHEN** the current ledger matches the journal's original digest and the synchronized temporary sibling matches the expected digest and passes full ledger validation
- **THEN** the temporary sibling replaces the ledger durably, the journal is removed durably, and the requested mutation continues from the recovered expected bytes

#### Scenario: Interruption after replacement
- **WHEN** the current ledger already matches the journal's expected digest after replacement but cleanup was interrupted
- **THEN** any matching temporary sibling and the journal are cleaned durably and the requested mutation continues from the current recovered bytes

#### Scenario: Concurrent mutation encounters recovery
- **WHEN** two writers reach a ledger with an unfinished journal
- **THEN** at most the lock holder performs recovery and mutation while the other writer receives the existing deterministic lock conflict without inspecting or changing partial state

### Requirement: Ambiguous recovery state is preserved for investigation
The application MUST NOT guess, delete, replace, or continue mutation when a journal is malformed, contains unsafe names, has inconsistent digests, references a missing or changed temporary sibling, describes a current ledger different from both known states, or would recover bytes that fail the same validation required for mutation. It SHALL return a stable actionable application error, retain the journal and relevant files, and leave the current ledger unchanged.

#### Scenario: Ledger changed outside the transaction
- **WHEN** the current ledger digest matches neither the original nor expected digest recorded in the unfinished journal
- **THEN** the mutation returns `conflict`, retains the journal and temporary sibling, and performs no ledger mutation

#### Scenario: Recovery candidate is invalid
- **WHEN** the temporary sibling matches the recorded expected digest but fails parse, schema, format, semantic, or retained-artifact validation
- **THEN** recovery fails without replacing the current ledger or deleting recovery evidence

### Requirement: Recovery behavior is documented and fault tested
Maintained troubleshooting and security documentation SHALL explain automatic journal recovery, the conditions that stop it, and the fact that users MUST NOT delete or edit recovery files as a normal remedy. Deterministic fault tests SHALL cover every stage before and after journal creation, replacement, directory synchronization, and cleanup on supported native filesystems.

#### Scenario: User retries after a process interruption
- **WHEN** a documented mutation is retried after a crash left a valid journal
- **THEN** the retry either recovers and completes safely or returns documented conflict guidance without requiring an undocumented internal function or manual journal deletion
