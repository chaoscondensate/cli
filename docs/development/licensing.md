# Check licensing compliance

<!-- doc-metadata
coverage: development
reviewed: 2026-08-25
owner: documentation
generated: false
security-critical: false
prerequisites: ../../LICENSE_POLICY.md, ../../REUSE.toml, ../../tools/docs/requirements.txt
next: documentation-style.md
-->

Forecast Ledger CLI uses the REUSE Specification 3.3 to keep copyright and
license information machine-readable. The project check is pinned to REUSE
tool 6.2.0 in `tools/docs/requirements.txt`.

## Prepare an isolated environment

Python 3.10 or newer is required. From the repository root:

```console
$ python3 -m venv .venv-docs
$ .venv-docs/bin/python -m pip install --requirement tools/docs/requirements.txt
```

On Windows PowerShell, use `.venv-docs\Scripts\python.exe` for the second
command. The installation contacts the Python package index. The compliance
check itself is local and does not require network access.

## Run the check

Activate the environment or call its executable directly:

```console
$ .venv-docs/bin/reuse lint
$ make license-check
```

The second command expects `reuse` to be available on `PATH`. A passing check
states that every covered repository file has machine-readable copyright and
license information and that all referenced license texts are present. It does
not audit whether a contributor had the right to submit third-party material.

## Update licensing data

When adding or changing a file:

1. use the default Apache-2.0 annotation for original project material;
2. add a later `REUSE.toml` override for copied or differently licensed work;
3. preserve upstream notices and add the material to
   `THIRD_PARTY_NOTICES.md` when it is distributed or needed for development;
4. add any new SPDX license text to `LICENSES/`; and
5. run `reuse lint` before review.

Do not install an unpinned REUSE version in CI. Review the release notes,
update the exact version in `tools/docs/requirements.txt`, and rerun the check
when upgrading it.

[Development documentation](index.md) · [Licensing policy](../../LICENSE_POLICY.md)
