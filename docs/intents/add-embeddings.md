# ADD_EMBEDDINGS

Attaches user data segments carrying Agentic Audiences embeddings.

**Payload:** `DataPayload` via the `data` field. Each segment **must** include `ext.aa`.

**Eligible paths:**
- `/user/data` — targets the user.

**Payload fields:**
- `data`: list of OpenRTB Data objects to attach to the user
- `data.name`: data provider name
- `data.segment`: segments from the provider
- `data.segment.id`: segment identifier
- `data.segment.name`: segment name
- `data.segment.ext.aa`: **required** Agentic Audiences embedding envelope
  - `ver` — embedding schema/spec version
  - `vector` — base64 Float32 little-endian (RFC 4648)
  - `dimension` — number of Float32 values
  - `model` — producing model identifier
  - `type` — signal types: `1`=identity, `2`=contextual, `3`=reinforcement

**Dependency:** OpenRTB protobuf `Segment.ext.aa` — [openrtb2.x#188](https://github.com/InteractiveAdvertisingBureau/openrtb2.x/pull/188).

## Example mutation

```json
{
  "intent": "ADD_EMBEDDINGS",
  "op": "OPERATION_ADD",
  "path": "/user/data",
  "content_data": {
    "data": [
      {
        "name": "data-provider",
        "segment": [
          {
            "id": "seg-ctx-001",
            "name": "descriptive-name",
            "ext": {
              "aa": {
                "ver": "1.0.0",
                "vector": "mpkZPq5HYb5SuJ4+PQrXPo/C9T7NzEy97FE4Pilcj74=",
                "dimension": 8,
                "model": "sbert-mini-ctx-001",
                "type": [2]
              }
            }
          }
        ]
      }
    ]
  }
}
```
