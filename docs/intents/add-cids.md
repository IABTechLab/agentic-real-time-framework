# ADD_CIDS

Attaches contextual segments containing extended content IDs.

**Payload:** `DataPayload` via the `content_data` field. Each `data` object must include `cids`.

**Eligible paths:**
- `/site/content/data` — targets site.
- `/app/content/data` — targets app. 

**Payload fields:**
- `data`: list of OpenRTB data objects to attach to content.
- `data.id`: data provider identifier.
- `data.name`: data provider name.
- `data.segment`: content identifiers or other data segments from that provider.
- `data.ext.cids`: content IDs for that content source. 

## Example mutation

```json
{
  "intent": "ADD_CIDS",
  "op": "OPERATION_ADD",
  "path": "/site/content/data",
  "content_data": {
    "data": [
      {
        "id": "provider-1",
        "name": "Content Provider",
        "ext": {
          "cids": ["content-abc-001", "content-abc-002"]
        }
      }
    ]
  }
}
```
