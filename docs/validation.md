# Validation

Kashvi's validation engine lives in `pkg/validate`. It has **zero external dependencies** and supports 28 rules via struct tags.

---

## Struct Tags

Add a `validate` tag to any field:

```go
type RegisterInput struct {
    Name            string  `json:"name"             validate:"required,min=2,max=100"`
    Email           string  `json:"email"            validate:"required,email"`
    Age             int     `json:"age"              validate:"required,min=18,max=120"`
    Role            string  `json:"role"             validate:"in=admin,user,editor"`
    Password        string  `json:"password"         validate:"required,min=8"`
    PasswordConfirm string  `json:"password_confirm" validate:"confirmed=password"`
    Website         *string `json:"website"          validate:"nullable,url"`
}
```

---

## All Validation Rules

| Rule | Example Tag | Description |
|---|---|---|
| `required` | `validate:"required"` | Field must be non-zero |
| `email` | `validate:"email"` | Valid email address |
| `min` | `validate:"min=3"` | String min length / numeric min value |
| `max` | `validate:"max=100"` | String max length / numeric max value |
| `between` | `validate:"between=1,10"` | Numeric between two values (inclusive) |
| `in` | `validate:"in=a,b,c"` | Value must be one of the listed options |
| `not_in` | `validate:"not_in=bad,worse"` | Value must NOT be in the list |
| `confirmed` | `validate:"confirmed=password"` | Must match another field's value |
| `url` | `validate:"url"` | Valid HTTP/HTTPS URL |
| `alpha` | `validate:"alpha"` | Letters only |
| `alpha_num` | `validate:"alpha_num"` | Letters and numbers only |
| `alpha_dash` | `validate:"alpha_dash"` | Letters, numbers, `-`, `_` |
| `numeric` | `validate:"numeric"` | Any number (int or float) |
| `integer` | `validate:"integer"` | Must be an integer |
| `boolean` | `validate:"boolean"` | true or false |
| `ip` | `validate:"ip"` | Valid IPv4 or IPv6 address |
| `uuid` | `validate:"uuid"` | Valid UUID |
| `date` | `validate:"date"` | Valid date in `YYYY-MM-DD` format |
| `date_format` | `validate:"date_format=2006-01-02"` | Custom Go time layout |
| `starts_with` | `validate:"starts_with=https"` | String prefix check |
| `ends_with` | `validate:"ends_with=.go"` | String suffix check |
| `contains` | `validate:"contains=@"` | Substring check |
| `regex` | `validate:"regex=^[A-Z]+"` | Custom regex pattern |
| `json` | `validate:"json"` | Valid JSON string |
| `len` | `validate:"len=6"` | Exact string length |
| `same` | `validate:"same=other_field"` | Alias for `confirmed` |
| `different` | `validate:"different=old_password"` | Must differ from field |
| `nullable` | `validate:"nullable,email"` | Skip all other rules if the field is nil/zero |

---

## Using Validation Directly

### In a handler with `BindJSON`:
```go
func (ctrl *UserController) Register(c *appctx.Context) {
    var input RegisterInput
    if !c.BindJSON(&input) {
        return // 422 already sent
    }
    // input is valid here
}
```

### Manual validation:
```go
import "github.com/shashiranjanraj/kashvi/pkg/validate"

errs := validate.Struct(&input)
if validate.HasErrors(errs) {
    // errs = map[string]string{"email": "The email field must be a valid email address."}
}
```

---

## Error Messages

Errors are returned as `map[string]string` where the key is the JSON field name:

```json
{
  "status": 422,
  "message": "Validation failed",
  "errors": {
    "email": "The email field must be a valid email address.",
    "password": "The password field must be at least 8 characters.",
    "password_confirm": "The password_confirm field must match password."
  }
}
```

---

## Nullable Fields

Use `nullable` to skip all other rules when the field is empty/nil:

```go
type UpdateInput struct {
    // These are all optional — only validated if provided
    Bio     *string `json:"bio"     validate:"nullable,max=500"`
    Website *string `json:"website" validate:"nullable,url"`
    Age     *int    `json:"age"     validate:"nullable,min=18"`
}
```

---

## Combining Rules

Rules are comma-separated and evaluated in order. All failures are collected (not short-circuit):

```go
validate:"required,min=8,max=64,alpha_num"
```
