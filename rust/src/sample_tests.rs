//! Tests that exercise the bundled sample payloads in `samples/`.
//!
//! Each sample must decode as a schema-valid `RtbRequest` (proto3 JSON
//! mapping, unknown fields rejected) and drive the demo evaluator to the
//! mutations its scenario describes. This keeps the shared sample payloads
//! and the reference implementation from drifting apart.

use prost_reflect::{DescriptorPool, DynamicMessage};

use crate::bidder::evaluate;
use crate::proto::com::iabtechlab::bidstream::mutation::v1::{mutation::Value, Intent, RtbRequest};

const DESCRIPTOR_BYTES: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/artf_descriptor.bin"));

const SAMPLES: [&str; 5] = [
    "banner-basic.json",
    "video-deals.json",
    "bid-shading.json",
    "multi-impression.json",
    "native-ad.json",
];

fn load_sample(name: &str) -> RtbRequest {
    let path = format!("{}/../samples/{}", env!("CARGO_MANIFEST_DIR"), name);
    let json = std::fs::read_to_string(&path)
        .unwrap_or_else(|error| panic!("failed to read {path}: {error}"));

    let pool = DescriptorPool::decode(DESCRIPTOR_BYTES).expect("descriptor set");
    let descriptor = pool
        .get_message_by_name("com.iabtechlab.bidstream.mutation.v1.RTBRequest")
        .expect("RTBRequest descriptor");

    let mut deserializer = serde_json::Deserializer::from_str(&json);
    let message = DynamicMessage::deserialize(descriptor, &mut deserializer)
        .unwrap_or_else(|error| panic!("{name} is not a valid RtbRequest payload: {error}"));
    deserializer.end().expect("trailing JSON content");

    message.transcode_to().expect("transcode to RtbRequest")
}

#[test]
fn all_samples_decode_as_rtb_requests() {
    for name in SAMPLES {
        let request = load_sample(name);
        assert!(!request.id.is_empty(), "{name} carries an envelope id");
        assert!(
            request.bid_request.is_some(),
            "{name} carries a bid_request"
        );
        assert!(
            !request.applicable_intents.is_empty(),
            "{name} declares applicable intents"
        );
    }
}

#[tokio::test]
async fn samples_drive_demo_evaluator_mutations() {
    // bid-shading.json declares only BID_SHADE applicable, so the demo
    // evaluator's segment and deal mutations for that auction are filtered out.
    let cases = [
        ("banner-basic.json", 2),
        ("video-deals.json", 2),
        ("bid-shading.json", 1),
        ("multi-impression.json", 4),
        ("native-ad.json", 2),
    ];
    for (name, expected_mutations) in cases {
        let request = load_sample(name);
        let envelope_id = request.id.clone();

        let response = evaluate(request).await;

        assert_eq!(response.id, envelope_id, "{name} echoes the envelope id");
        assert_eq!(
            response.mutations.len(),
            expected_mutations,
            "{name} produces the demo mutations for its scenario"
        );
        assert!(response.metadata.is_some(), "{name} carries metadata");
    }
}

#[tokio::test]
async fn bid_shading_sample_produces_bid_shade_mutation() {
    let response = evaluate(load_sample("bid-shading.json")).await;
    let shade = response
        .mutations
        .iter()
        .find(|mutation| mutation.intent == Intent::BidShade as i32)
        .expect("bid shade mutation");
    match shade.value.as_ref() {
        Some(Value::AdjustBid(adjust)) => assert!(adjust.price > 0.0),
        other => panic!("expected AdjustBid payload, got {other:?}"),
    }
}
