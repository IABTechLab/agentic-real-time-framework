# ADD_METRICS

Attaches additional OpenRTB metrics to an impression, such as viewability or audibility.

**Payload:** `MetricsPayload` via the `metrics` field

**Eligible paths:**
- `/imp/{id}` — targets a specific impression by ID.

**Payload fields:**
- `metric`: list of OpenRTB metric objects to attach to the impression.
- `metric.type`: metric name, such as `viewability`.
- `metric.value`: metric value, typically a 0.0 to 1.0 score.
- `metric.vendor`: source of the metric value.

## Example mutation

```json
{
  "intent": "ADD_METRICS",
  "op": "OPERATION_ADD",
  "path": "/imp/{id}",
  "metrics": {
    "metric": [
      {
        "type": "viewability",
        "value": 0.93,
        "vendor": "EXCHANGE"
      },
      {
        "type": "audibility",
        "value": 0.81,
        "vendor": "EXCHANGE"
      }
    ]
  }
}
```
