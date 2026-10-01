# ADJUST_DEAL_FLOOR

Raises or lowers the floor on a specific deal while keeping the rest of the deal targeting intact.

**Payload:** `AdjustDealPayload` via the `adjust_deal` field

**Eligible paths:**
- `/imp/{id}/deals/{dealId}` — targets a specific deal within a specific impression.

**Payload fields:**
- `bidfloor`: new floor value to apply to the deal.

## Example mutation

```json
{
  "intent": "ADJUST_DEAL_FLOOR",
  "op": "OPERATION_REPLACE",
  "path": "/imp/{id}/deals/{dealId}",
  "adjust_deal": {
    "bidfloor": 5.5
  }
}
```
