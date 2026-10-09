# BID_SHADE

Reduces the bid price for a bid response.

**Payload:** `AdjustBidPayload` via the `adjust_bid` field

**Eligible paths:**
- `/seatbid/{seat}/bid/{bidId}` — targets a specific bid within a specific seat bid.

**Payload fields:**
- `price`: bid price to write into the bid response.

**Restrictions:**
- The adjusted price must be lower than the original bid.
- This intent only applies to a bid response value that already exists in the auction.

## Example mutation

```json
{
  "intent": "BID_SHADE",
  "op": "OPERATION_REPLACE",
  "path": "/seatbid/{seat}/bid/{bidId}",
  "adjust_bid": {
    "price": 4.25
  }
}
```
