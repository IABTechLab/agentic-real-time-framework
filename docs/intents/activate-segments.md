# ACTIVATE_SEGMENTS

Activates segments for the current bid request.

**Payload:** `IDsPayload` via the `ids` field

**Eligible paths:**
- `/user/data/segment` — target users.

**Payload fields:**
- `id`: external segment IDs to activate for the request.

## Example mutation

```json
{
  "intent": "ACTIVATE_SEGMENTS",
  "op": "OPERATION_ADD",
  "path": "/user/data/segment",
  "ids": {
    "id": ["sports-fans", "premium-subscribers"]
  }
}
```
