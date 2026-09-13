# iSolarCloud API

Status: Normative  
Scope: Session construction, wire protocol, endpoint catalog, caching, and gateway recovery

## Gateway and authentication

- **REQ-API-001** — The default gateway MUST be `https://augateway.isolarcloud.com`; the normal request timeout MUST be bounded at 60 seconds and the product-level API default at 30 seconds.
- **REQ-API-002** — The current, old, and legacy client app keys are respectively `B0455FBE7AA0328DB57B59AA729F05D8`, `A5C22A880B97303FCB902069C6B042AB`, and `93D72E60331ABDCDC7B39ADC2D1F32B3`. Empty or legacy configured keys MUST normalize to the current key; an explicitly configured old or custom key is retained.
- **REQ-API-003** — Login requires non-empty app key, username, and password. It MUST send `login_type=1`, `strong_weak_password=1`, `rememberMe=false`, `supportTotp=1`, `isNamePassword=true`, and login `sys_code=900`.
- **REQ-API-004** — A cached token is valid only when the response reports success, the token is non-empty, and its login time is no more than 24 hours old. Forced login MUST invalidate the cached token before calling the endpoint.
- **REQ-API-005** — Missing token files are cache misses. Syntactically corrupt token files MUST be removed and treated as cache misses. Other token-file I/O failures MUST be reported.
- **REQ-API-006** — A successful fresh login MUST persist the full authentication response, retain auth details for recovery, and expose token, user ID, last-login time, and app key internally. Printed auth diagnostics MUST redact the token.
- **REQ-API-007** — Login attempts MUST try unique `(host, app-key)` pairs in nested order: configured host, `https://augateway.isolarcloud.com`, `https://gateway.isolarcloud.com`, `https://gateway.isolarcloud.eu`, `https://gateway.isolarcloud.com.hk`, `https://gateway.isolarcloud.com.cn`; for each host use normalized configured key, current key, old key, then legacy key. Empty pairs and exact duplicates are omitted while preserving first occurrence.
- **REQ-API-008** — A next login candidate MAY be attempted only for a recoverable gateway error and MUST NOT be attempted for Docker resolver errors involving `127.0.0.11:53`.
- **REQ-API-009** — Exhausted candidate errors MUST be summarized by host in attempt order, with attempt count and at most two distinct messages per host.

## Wire protocol

- **REQ-API-010** — Endpoint requests MUST use HTTP POST to the selected gateway plus endpoint path and MUST serialize endpoint-specific data with common fields `appkey`, `lang`, `sys_code`, optional `token`, optional `user_id`, and optional `valid_flag`.
- **REQ-API-011** — Empty optional auth fields MUST be omitted. Missing language defaults to `_en_US`; normal endpoint `sys_code` is numeric `200`; missing app key uses the current default.
- **REQ-API-012** — Every request MUST include `api_key_param` containing a timestamp in milliseconds and a random 32-character nonce. The timestamp uses local current time plus the learned server offset and the optional diagnostic offset defined by `REQ-CFG-004`.
- **REQ-API-013** — The compact JSON body MUST be encrypted with AES using a random 32-character key beginning with `web`; the key MUST be RSA PKCS#1 v1.5 encrypted into `x-random-secret-key`. Responses MUST be decrypted with the same AES key when decryption yields valid JSON; otherwise a valid plaintext JSON body MAY be used.
- **REQ-API-014** — Required browser-compatible headers are `accept=application/json, text/plain, */*`, `accept-language=en-US,en;q=0.9`, a Chrome 132 Windows user agent, `origin=https://www.isolarcloud.com`, `referer=https://www.isolarcloud.com/`, `content-type=text/plain;charset=UTF-8`, `sys_code=200`, `_pl=js`, `_did`, `_global_new_web=1`, `_vc=2026011301`, `_browser_brand=chrome`, `_browser_version=132.0`, `x-client-tz`, `x-sign-code=0`, and `x-access-key=9grzgbmxdsp3arfmmgq347xjbza4ysps`. `_did` follows `REQ-CFG-004`; for wire compatibility `x-client-tz` is `GMT%2B<hours>` for a nonnegative local UTC offset or `GMT-<absolute-hours>` for a negative offset, using the truncated whole-hour component.
- **REQ-API-015** — When a token exists, its prefix before `_` MUST be RSA-encrypted as `x-limit-obj`; `GOSUNGROW_LIMIT_OBJ` overrides it, and `__EMPTY__` explicitly requests an encrypted empty value.
- **REQ-API-016** — HTTP 401 and every non-200 response MUST fail. Empty bodies MUST fail. Transport tracing MAY be enabled by `GOSUNGROW_TRACE_HTTP`, but production operation SHOULD keep it disabled because request payloads can contain secrets.
- **REQ-API-017** — On an “Expired request” response, the client MUST retry at most once after deriving a clock offset from the HTTP `Date` header, then reset retry state.

## Response contract

- **REQ-API-018** — Common response fields are `req_serial_num`, `result_code`, and `result_msg`. Success requires message empty or `success`, code `1`, and non-empty request serial.
- **REQ-API-019** — Codes `-1`, `010`, `000`, and `201` are generic errors; `E00003` means login required; `E914` means gateway login rejection; `E917` means a required request header is missing; unknown codes are errors.
- **REQ-API-020** — Messages `Expired request`, `er_invalid_appkey`, `er_token_login_invalid`, `er_parameter_value_invalid`, `er_unknown_exception`, and strings starting `Parameter:` MUST map respectively to clock-skew, app-key, login-required, request-data, API, and request-data errors. Unknown non-empty messages are errors.

## Endpoint catalog

All endpoints MUST validate required fields, preserve unknown response fields where feasible, convert known fields through [data-normalization.md](data-normalization.md), and use the common response contract.

| Logical endpoint | Path | Required request | Primary result | Cache |
|---|---|---|---|---|
| `AppService.login` | `/v1/userService/login` | account, password | auth/session/user fields | token validity |
| `AppService.getUserList` | `/v1/userService/getUserList` | none | users | normal default |
| `AppService.getPsList` | `/v1/powerStationService/getPsList` | none | plants | 5 min |
| `AppService.getPsDetail` | `/v1/powerStationService/getPsDetail` | `ps_id` | plant detail and virtual fields | caller-defined |
| `AppService.getDeviceList` | `/v1/devService/getDeviceList` | `ps_id` | device page list | 24 h |
| `AppService.queryDeviceList` | `/v1/devService/queryDeviceList` | `ps_id` | devices and point data | sync interval |
| `queryDeviceRealTimeDataByPsKeys` | routed to `/v1/devService/queryDeviceList` | `ps_key_list` public input | device point data | sync interval |
| `WebAppService.getDevicePointAttrs` | `/v1/devService/getDevicePointAttrs` | `deviceType`, `psId`; optional `uuid` | point metadata | 24 h |
| `WebIscmAppService.getPsTreeMenu` | `/v1/devService/getPsTreeMenu` | `ps_id` | plant device tree | 1 h |

- **REQ-API-021** — The realtime public contract accepts `ps_key_list`, derives `ps_id` from the substring before the first `_`, uses a number when parseable and otherwise the string, removes `ps_key_list`, and calls the query-device-list path.
- **REQ-API-022** — Plant IDs supplied by the caller MUST be used; otherwise they MUST be discovered from `getPsList`. Plant tree queries occur once per plant.
- **REQ-API-023** — Device point metadata MUST be fetched for every device in each requested plant tree using its UUID, plant ID, and device type; an optional device-type filter is applied after retrieval.
- **REQ-API-024** — Plant keys SHOULD come from the plant tree; device-list keys are the fallback when the tree returns none. Keys MUST be unique and non-empty.

## Cache and endpoint replay

- **REQ-API-025** — Endpoint cache identity MUST include endpoint area/name plus a deterministic fingerprint of endpoint-specific request data.
- **REQ-API-026** — Fresh cache is used before the network. Unreadable or syntactically corrupt cached JSON MUST be removed and fetched once from the API. Invalid response data MUST remove its cache entry.
- **REQ-API-027** — A successful uncached response MUST be cached using replacement-safe file writing. Cache write failure is an operation failure.
- **REQ-API-028** — A recoverable endpoint failure MAY cause exactly one session recovery and endpoint replay. Replay MUST retain endpoint name, request JSON, and cache timeout while using the recovered gateway/session.
- **REQ-API-029** — Endpoint replay MUST NOT occur when logged out, already recovering, error is non-recoverable, error is Docker DNS, or auth details are unavailable.
- **REQ-API-030** — RSA encryption MUST use the URL-safe base64 DER public key `MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCkecphb6vgsBx4LJknKKes-eyj7-RKQ3fikF5B67EObZ3t4moFZyMGuuJPiadYdaxvRqtxyblIlVM7omAasROtKRhtgKwwRxo2a6878qBhTgUVlsqugpI_7ZC9RmO2Rpmr8WzDeAapGANfHN5bVr7G7GYGwIrjvyxMrAVit_oM4wIDAQAB`. These fixed app/access/public keys are protocol client identifiers embedded in the public product; user passwords and session tokens remain secrets.

## Prohibited behavior

- Sending empty tokens/user IDs as meaningful values.
- Logging plaintext tokens, passwords, AES keys, or decrypted credentials.
- Reusing an expired request indefinitely.
- Retrying a Docker embedded-DNS failure across gateway hosts.
