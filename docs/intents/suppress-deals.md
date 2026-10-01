# SUPPRESS_DEALS

Suppresses specific deals from consideration, for example because of brand-safety, policy, pacing, or campaign constraints.

**Payload:** `IDsPayload` via the `ids` field

**Eligible paths:**
- `/imp/{id}` — targets a specific impression by ID.

**Payload fields:**
- `id`: external deal IDs to suppress from eligibility.

## Example mutation

```json
{
  "intent": "SUPPRESS_DEALS",
  "op": "OPERATION_REMOVE",
  "path": "/imp/{id}",
  "ids": {
    "id": ["deal-archived", "deal-blocked"]
  }
}
```
