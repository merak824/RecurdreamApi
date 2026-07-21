# Payment Channel Fees and EliandaPay V2 Design

## Status

Draft for written review. The architecture and fee policy were approved in conversation on 2026-07-21.

## Summary

This change adds two payment capabilities without changing the behavior of existing EasyPay V1 provider instances:

1. Optional Alipay and WeChat recharge fee overrides, with the existing global recharge fee as the fallback.
2. A separate EliandaPay V2 provider using SHA256WithRSA, V2 payment endpoints, signed callbacks, order queries, refunds, refund queries, and order closing.

The two features share the existing payment order, provider-instance, load-balancing, and fulfillment infrastructure. Existing EasyPay V1 orders and credentials remain valid.

## Goals

- Let administrators configure different user-facing surcharge rates for Alipay and WeChat.
- Keep the current user-friendly surcharge formula: the displayed fee is calculated from the requested recharge or subscription amount.
- Apply the selected payment method's effective fee to both balance recharges and subscription purchases.
- Keep the current global recharge fee as the fallback for unset method overrides and other payment methods.
- Add EliandaPay V2 as a new provider that can coexist with EasyPay V1.
- Verify V2 responses and callbacks with the platform public key before accepting payment state.
- Support the existing optional provider capabilities for refund status queries and closing unpaid orders.
- Preserve the current order fulfillment, refund, provider pinning, and webhook acknowledgement behavior.

## Non-Goals

- Do not remove, migrate, or reinterpret existing EasyPay V1 provider instances.
- Do not add EliandaPay transfer/payout APIs, merchant-list APIs, balance APIs, or settlement-management APIs.
- Do not add a general expression engine for payment fees.
- Do not attempt to make the surcharge exactly offset every cent of upstream settlement cost. The platform accepts the small fee-on-fee difference so the user sees the advertised percentage.
- Do not change storage encryption for payment-provider configuration as part of this feature.

## Current Behavior

The current system has one global recharge fee setting. Order creation reads that value before selecting the payment provider and stores it in `payment_orders.fee_rate`. The frontend also uses the global value when previewing the amount due.

The current EasyPay provider implements the legacy V1 protocol:

- MD5 signing with `PID + PKey`
- `/submit.php` and `/mapi.php` for payment creation
- `/api.php` for queries and refunds
- MD5 verification for payment callbacks

EliandaPay V2 is a different protocol and cannot be enabled by inserting an RSA private key into the V1 `PKey` field.

## Design: Per-Method Recharge Fees

### Settings

Keep the existing global setting:

- Internal key: `RECHARGE_FEE_RATE`
- API field: `payment_recharge_fee_rate`

Add two optional override settings:

- Internal key: `ALIPAY_RECHARGE_FEE_RATE`
- API field: `payment_alipay_fee_rate`
- Internal key: `WXPAY_RECHARGE_FEE_RATE`
- API field: `payment_wxpay_fee_rate`

The override values are nullable. Their semantics are:

- `null` or an unset/empty stored value: use the global recharge fee.
- `0`: explicitly charge no fee for that payment method.
- A value greater than `0`: use that percentage for that payment method.

All fee values allow at most two decimal places and must be between `0` and `100` inclusive.

### Resolution Rule

Order creation resolves one effective rate from the user-facing payment type:

1. For `alipay`, use the Alipay override when configured.
2. For `wxpay`, use the WeChat override when configured.
3. For all other payment types, or when the corresponding override is unset, use the global recharge fee.

Custom EasyPay methods do not inherit Alipay or WeChat overrides merely because their upstream type maps to `alipay` or `wxpay`. They use the global fee unless a future feature explicitly adds custom-method fee configuration.

### Amount Calculation

Preserve the current additive calculation and currency rounding:

```text
fee amount = requested amount * effective fee rate / 100
pay amount = requested amount + fee amount
```

The user-facing preview, backend order calculation, and stored order values must use the same rounding helper. For example:

```text
Requested balance: CNY 10.00
Alipay fee rate: 3.00%
Pay amount: CNY 10.30
Credited balance: unchanged by the fee
```

The platform deliberately absorbs any upstream fee-on-fee settlement difference. It does not raise the displayed fee above the configured percentage.

### User Interface

Add two inputs beside the existing global recharge fee setting:

- Alipay fee rate
- WeChat fee rate

An empty input displays a hint that the global recharge fee will be used. Entering `0` explicitly disables the fee for that method.

The checkout endpoint must expose the effective `fee_rate` for each available payment method. The payment page must calculate its preview from the currently selected method's rate, with the global value used only as a compatibility fallback.

### Backward Compatibility

Existing deployments have no values for the two new settings. They therefore continue using the current global fee for every method. Existing orders retain their stored fee rate and are not recalculated.

## Design: EliandaPay V2 Provider

### Provider Identity

Add a new provider key:

```text
eliandapay_v2
```

It is separate from the existing `easypay` key. Provider selection, order snapshots, pending-order protections, webhook verification, and refund operations continue using the existing provider-instance abstractions.

### Configuration

The provider requires:

- `pid`: EliandaPay merchant ID
- `merchantPrivateKey`: merchant RSA private key used to sign requests
- `platformPublicKey`: EliandaPay platform RSA public key used to verify responses and callbacks
- `apiBase`: defaults to `https://api.ndow.cn`
- `notifyUrl`: generated from the deployment origin and the V2 webhook path
- `returnUrl`: generated from the deployment origin and `/payment/result`

The private key and platform public key are sensitive provider fields and must be omitted from admin read responses. The merchant ID and both keys cannot be changed while the provider instance has pending orders.

The provider supports the user-facing payment types `alipay` and `wxpay`.

### Webhook Route

Register a dedicated public callback route:

```text
GET  /api/v1/payment/webhook/eliandapay-v2
POST /api/v1/payment/webhook/eliandapay-v2
```

The callback handler reuses the shared payment webhook pipeline but resolves and verifies only `eliandapay_v2` provider instances. A successfully handled callback returns plain text `success`.

### Request Signing

V2 requests use SHA256WithRSA with PKCS#1 v1.5 padding:

1. Remove `sign`, `sign_type`, empty values, arrays, and binary values.
2. Sort parameter names in ascending byte/ASCII order.
3. Join entries as `key=value` with `&` separators, without URL-encoding values in the signing string.
4. Sign the UTF-8 bytes with the merchant private key and SHA-256.
5. Base64-encode the signature into `sign` and send `sign_type=RSA`.

The implementation accepts standard PEM keys. It may normalize a raw base64 key body by adding the expected PEM header and footer, but it must reject malformed or unparsable keys at provider save time.

### Response and Callback Verification

Successful V2 API responses must contain a valid platform signature. Error responses without a signature may be parsed only to surface their error code and message; they cannot update order state.

Callbacks must pass all of these checks:

- Valid SHA256WithRSA signature using `platformPublicKey`
- Matching configured `pid`
- Non-empty `out_trade_no`
- `trade_status` equal to `TRADE_SUCCESS` before producing a success notification
- A valid numeric amount
- A timestamp close enough to the server clock to reject stale replayed callbacks

Use a five-minute timestamp tolerance. Operators must keep the server clock synchronized. Signature verification includes all non-empty extension fields supplied by the platform so future callback fields do not break validation.

### Payment Creation

Call:

```text
POST {apiBase}/api/pay/create
Content-Type: application/x-www-form-urlencoded
```

Send the existing Sub2API order ID as `out_trade_no`, the selected method as `type`, the calculated `pay_amount` as `money`, and the standard notification and return URLs.

The initial implementation uses `method=jump`. This is the stable V2 flow documented to return a hosted payment URL and avoids introducing unsafe HTML rendering or incomplete JSAPI/APP handling. Map a successful `pay_type=jump` response to the existing `PayURL` field.

If the platform returns another `pay_type`, fail with a clear unsupported-response error rather than guessing how to render it. QR-code, JSAPI, APP, scan, and applet-specific V2 flows can be added separately when the product has corresponding UX requirements.

Do not send `fee_mode=1`. Sub2API already calculates and displays the user surcharge. Enabling upstream buyer-fee mode would risk charging the user twice. Leave the field absent so the merchant's normal EliandaPay settlement configuration applies.

### Order Query

Call:

```text
POST {apiBase}/api/pay/query
```

Query by `out_trade_no`, verify a successful response signature, and map status values:

- `0` to pending
- `1` to paid
- `2` to refunded
- Other values to the closest existing provider status without completing an unpaid order

Return the platform `trade_no`, amount, and merchant metadata through the existing `QueryOrderResponse` contract.

### Refund

Call:

```text
POST {apiBase}/api/pay/refund
```

Prefer `out_trade_no` when the Sub2API order ID is available and otherwise use the platform `trade_no`. Send the requested refund amount and verify the response signature before reporting success. The current provider refund contract does not supply a merchant refund number, so the initial V2 implementation leaves optional `out_refund_no` unset.

EliandaPay V2 uses `code=0` for success. This behavior is provider-specific and must not change the existing EasyPay V1 response handling.

### Refund Query

Implement the existing `RefundQueryProvider` extension by calling:

```text
POST {apiBase}/api/pay/refundquery
```

Use the platform refund number when it is available, otherwise query by the original merchant or platform order identifier according to the V2 API contract. Verify the response signature and map the upstream refund state to the existing `success`, `pending`, or `failed` refund statuses.

### Close Unpaid Order

Implement the existing `CancelableProvider` extension by calling:

```text
POST {apiBase}/api/pay/close
```

The payment lifecycle passes the merchant order number to `CancelPayment`, so send it as `out_trade_no`. Treat a signed `code=0` response as success. If the upstream reports that the order is already paid or cannot be closed, return an error and let the normal order query and reconciliation path determine final state.

### Timeouts and Response Limits

Use the existing payment-provider conventions:

- Ten-second upstream HTTP timeout
- One-megabyte maximum response body
- Form-encoded requests
- JSON responses
- Errors that include the provider operation and a bounded upstream message summary

Never log private keys, public-key bodies, signatures, or complete callback payloads at normal log levels.

## Data Flow

### Checkout

1. The user selects Alipay or WeChat and enters a recharge amount or chooses a subscription.
2. The frontend reads the selected method's effective fee and displays the base amount, fee, and pay amount.
3. The backend independently resolves the same fee rate and calculates the authoritative pay amount.
4. The load balancer selects an enabled provider instance for the payment type.
5. For `eliandapay_v2`, the provider signs and submits the V2 create request.
6. The frontend opens the returned hosted checkout URL.

### Callback

1. EliandaPay calls the dedicated V2 webhook with the merchant order number.
2. The webhook pipeline loads the order's pinned V2 provider instance.
3. The provider verifies the platform signature, timestamp, merchant ID, status, and amount.
4. The payment service idempotently marks and fulfills the order.
5. The endpoint returns `success`.

## Error Handling

- Reject invalid fee settings before they are persisted.
- Treat missing V2 keys or invalid PEM data as provider configuration errors at save time.
- Reject successful but unsigned V2 responses.
- Reject callbacks with invalid signatures, stale timestamps, mismatched merchant IDs, invalid amounts, or unsupported statuses.
- Preserve idempotent callback handling and acknowledge unknown orders according to the existing webhook policy.
- Surface unsupported V2 `pay_type` values as explicit gateway errors.
- Do not fall back from V2 to V1 credentials or verification logic.

## Testing Strategy

### Backend Fee Tests

- Alipay override replaces the global fee.
- WeChat override replaces the global fee.
- An explicit zero override disables the method fee.
- An unset override falls back to the global fee.
- Other methods use the global fee.
- Balance and subscription orders both store and charge the resolved effective rate.
- Existing global-only configurations keep their current behavior.

### Frontend Fee Tests

- Settings inputs preserve null as "use global" and zero as an explicit override.
- Switching from Alipay to WeChat changes the displayed fee and pay amount.
- Checkout previews match backend additive rounding.
- Existing checkout payloads without per-method fee values continue using the global fee.

### V2 Provider Tests

- Canonical parameter ordering and empty-field exclusion.
- RSA signing verified with a generated public key.
- Platform signatures accepted for valid responses and rejected after tampering.
- Provider creation rejects missing or malformed key material.
- Create requests use the correct endpoint, form fields, timestamp, and signature.
- Create responses map `pay_type=jump` and reject unsupported types.
- Query responses map pending, paid, and refunded states correctly.
- Refund requests and success responses use V2 `code=0` semantics.
- Refund-query responses map upstream states to existing refund statuses.
- Closing an unpaid order sends `out_trade_no` to `/api/pay/close` and verifies the response.
- Valid callbacks produce a success notification.
- Invalid, stale, wrong-PID, and tampered callbacks fail verification.
- The V1 EasyPay signing, creation, query, refund, and webhook test suites remain green.

### Integration Tests

- Admin provider APIs can create and update an EliandaPay V2 instance while masking both keys.
- The frontend displays the V2 provider configuration fields and callback URL.
- Webhook routing reaches the V2 verifier for both GET and POST callbacks.
- Existing EasyPay V1 instances remain selectable and functional after the new provider is registered.

## Deployment and Rollback

No database schema migration is required. New fee values use the existing settings store, and V2 credentials use the existing generic provider-instance configuration column.

Deployment order:

1. Deploy backend and frontend together.
2. Leave the V2 provider disabled while entering and validating credentials.
3. Configure Alipay and WeChat fee overrides.
4. Enable the V2 provider and run a small real payment.
5. Verify callback fulfillment and order query reconciliation before increasing traffic.

Rollback is configuration-driven: disable the V2 provider and clear the per-method fee overrides. Existing V1 instances and the global fee remain available.
