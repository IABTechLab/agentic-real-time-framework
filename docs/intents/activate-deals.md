# ACTIVATE_DEALS

Activates a bid request for a set of deal IDs.

**Payload:** `IDsPayload` via the `ids` field

**Eligible paths:**
- `/imp/{id}` — targets a specific impression (`imp`) by ID.

**Payload fields:**
- `id`: external deal IDs to activate on the impression.

## Example mutation

```json
{
  "intent": "ACTIVATE_DEALS",
  "op": "OPERATION_ADD",
  "path": "/imp/{id}",
  "ids": {
    "id": ["deal-001", "deal-002"]
  }
}
```
