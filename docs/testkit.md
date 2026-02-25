# TestKit — JSON-Scenario-Driven API Testing

`pkg/testkit` lets you write REST API integration tests **entirely in JSON**. One JSON file = one test case. No repeated Go boilerplate.

It is powered by [testify](https://github.com/stretchr/testify) — `testify/assert` for assertions and `testify/mock` for mocking side-effects.

---

## Concept

```
testdata/
  create_user.json          ← scenario (what to do & assert)
  create_user_req.json      ← request body
  create_user_res.json      ← expected response body
  health_check.json         ← another scenario
```

One Go test function runs all of them:

```go
func TestAPI(t *testing.T) {
    handler := kernel.NewHTTPKernel().Handler()
    testkit.RunDir(t, handler, "testdata")
}
```

---

## Scenario JSON schema

```json
{
  "name":             "Create User",
  "description":      "POST /api/v1/users returns 201",
  "requestMethod":    "POST",
  "requestUrl":       "/api/v1/users",
  "requestFileName":  "create_user_req.json",
  "responseFileName": "create_user_res.json",
  "expectedCode":     201,
  "isMockRequired":   true,
  "isDbMocked":       false,
  "headers": {
    "Authorization": "Bearer test-token"
  },
  "netUtilMockStep": [
    {
      "method":    "httprequest",
      "isMock":    true,
      "matchUrl":  "https://verify.external.com/",
      "returnData": { "statusCode": 200, "body": "eyJ2ZXJpZmllZCI6dHJ1ZX0=" }
    },
    {
      "method":    "sendmail",
      "isMock":    true,
      "returnData": { "body": "" }
    }
  ]
}
```

### Field reference

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | **Required.** Test name (shown in `go test -v` output) |
| `description` | string | Human-readable description |
| `requestMethod` | string | HTTP method. Default: `GET` |
| `requestUrl` | string | **Required.** URL path to call (e.g. `/api/v1/users`) |
| `requestFileName` | string | Path to request body JSON file (relative to scenario dir) |
| `responseFileName` | string | Path to expected response JSON file (relative to scenario dir) |
| `expectedCode` | int | **Required.** Expected HTTP status code |
| `isMockRequired` | bool | If `true`, any un-mocked outgoing call fails the test |
| `isDbMocked` | bool | Informational flag — reserved for DB mock wiring |
| `headers` | object | Extra request headers (e.g. auth tokens) |
| `netUtilMockStep` | array | List of mock steps (see below) |

---

## Mock steps

### HTTP request mock (`method: "httprequest"`)

Intercepts outgoing calls made via `pkg/http`. Matched by URL **prefix**.

```json
{
  "method":   "httprequest",
  "isMock":   true,
  "matchUrl": "https://api.stripe.com/",
  "returnData": {
    "statusCode": 200,
    "body": "eyJpZCI6ImNoXzEyMyJ9"
  }
}
```

- `matchUrl` — prefix match. Empty string matches **any** URL.
- `returnData.body` — **base64-encoded** response body.
- `returnData.statusCode` — defaults to `200`.

### Function mock (`method: "sendmail"` / `"sms"` / `"notification"`)

Intercepts non-HTTP side-effects. Built-in methods:

| Method | Intercepts |
|--------|-----------|
| `sendmail` | `pkg/mail` sends |
| `sms` | SMS/notification sends |
| `notification` | Push notification sends |

```json
{ "method": "sendmail", "isMock": true, "returnData": { "body": "" } }
```

### Custom function mock

Register your own mocker once in a test init:

```go
func init() {
    testkit.RegisterMocker("payments", testkit.NewFuncMocker("payments"))
}
```

Then use in JSON: `"method": "payments"`.

---

## Base64 encoding the body

```bash
# Encode: {"verified":true}
echo -n '{"verified":true}' | base64
# → eyJ2ZXJpZmllZCI6dHJ1ZX0=
```

---

## Runner API

```go
// Run a single scenario
testkit.Run(t, handler, "testdata/create_user.json")

// Run all *.json files in a directory as subtests
testkit.RunDir(t, handler, "testdata")
```

**Lifecycle per scenario:**
1. Load scenario JSON
2. Read request body from `requestFileName`
3. Install HTTP mock transport (`MockTransport`)
4. Activate function mocks (`sendmail`, `sms`, …)
5. Fire request against handler via `httptest`
6. Assert HTTP status code
7. JSON deep-diff actual vs expected response
8. Verify all `isMock: true` steps were called
9. Reset all mocks

---

## Advanced: testify mock expectations

Access the underlying `testify/mock.Mock` for custom assertions:

```go
func TestCreateUser(t *testing.T) {
    // Override the sendmail mocker
    mailer := testkit.NewFuncMocker("sendmail")
    mailer.Mock().On("Intercept", mock.Anything).Return(nil)
    testkit.RegisterMocker("sendmail", mailer)

    testkit.Run(t, handler, "testdata/create_user.json")

    // Assert it was called exactly once
    mailer.Mock().AssertNumberOfCalls(t, "Intercept", 1)
}
```

---

## Assertions

| Assertion | Behaviour |
|-----------|-----------|
| Status code | `testify/assert.Equal` — prints expected vs actual |
| Response body | JSON normalised (key order / whitespace ignored), `testify/assert.Equal` |
| HTTP mocks called | Fails per un-triggered `isMock: true` httprequest step |
| Func mocks called | Fails per un-triggered `isMock: true` func step |

---

## Debugging

Print a scenario summary to stdout:

```go
s, _ := testkit.LoadScenario("testdata/create_user.json")
testkit.DumpScenario(s)
```

Output:
```
Scenario: Create User
  POST /api/v1/users → 201
  requestFile:  create_user_req.json
  responseFile: create_user_res.json
  isMockRequired: true  isDbMocked: false
  mockStep[0]: method=httprequest  isMock=true  matchUrl="https://verify.external.com/"
  mockStep[1]: method=sendmail     isMock=true  matchUrl=""
```
