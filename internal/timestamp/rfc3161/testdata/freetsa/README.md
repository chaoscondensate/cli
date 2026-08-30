# FreeTSA RFC 3161 interoperability fixture

These byte-exact files contain no private forecast data. They were retrieved on
2026-08-30 from the qualified `https://freetsa.org/tsr` profile with OpenSSL
3.6.0:

```sh
openssl ts -query -sha256 -cert -data target.txt -out request.tsq
curl --fail --silent --show-error --max-redirs 0 \
  --header 'Content-Type: application/timestamp-query' \
  --header 'Accept: application/timestamp-reply' \
  --data-binary '@request.tsq' \
  --output response.tsr \
  https://freetsa.org/tsr
openssl ts -verify \
  -queryfile request.tsq \
  -in response.tsr \
  -CAfile ../../providers/freetsa/ca.pem \
  -untrusted tsa.pem
```

OpenSSL reports `Verification: OK`. The response uses a SHA-256 message
imprint, matching nonce, ESS `SigningCertificate` v1, SHA-512 CMS digest, and
ECDSA-with-SHA-512 signature.

| File | SHA-256 |
| --- | --- |
| `target.txt` | `0ac2fd85615071ded99db80f41e33d0ac879af474c7a5b39793c95bfb925fc5f` |
| `request.tsq` | `de76a83e68832390a86ffabd8c8b1227fb44259caa2b5af54e431b72a9f85611` |
| `response.tsr` | `af9cfc38e47957ab71e85ae88baf2b5cbe9acdfcd645f16b46099960717dd4a4` |
| `tsa.pem` | `8bfb0305bb64e2571ca507552ef3245cb1c2fee8728e0ff8689225081ea13467` |

The exact root bundle is
`internal/timestamp/rfc3161/providers/freetsa/ca.pem`, SHA-256
`2151b61137ffa86bf664691ba67e7da0b19f98c758e3d228d5d8ebf27e044438`.
Normal tests use these retained bytes and never contact the live service.
