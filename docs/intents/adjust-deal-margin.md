# ADJUST_DEAL_MARGIN

Adjusts the margin used for a deal.

**Payload:** `AdjustDealPayload` via the `adjust_deal` field

**Eligible paths:**
- `/imp/{id}/deals/{dealId}` — targets a specific deal within a specific impression.

**Payload fields:**
- `margin`: margin adjustment object for the deal.
- `margin.value`: the amount of margin to apply.
- `margin.calculation_type`: `CPM` for absolute margin or `PERCENT` for relative margin.

## Example mutation

```json
{
  "intent": "ADJUST_DEAL_MARGIN",
  "op": "OPERATION_REPLACE",
  "path": "/imp/{id}/deals/{dealId}",
  "adjust_deal": {
    "margin": {
      "value": 0.15,
      "calculation_type": "PERCENT"
    }
  }
}
```
