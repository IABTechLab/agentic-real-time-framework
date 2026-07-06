//! Bid evaluation logic that generates mutation responses.

use crate::mutation::builder::{activate_deals, activate_segments, bid_shade, build_metadata};
use crate::mutation::types::{
    PATH_IMP_1, PATH_IMP_FOOTER, PATH_IMP_HEADER, PATH_IMP_SIDEBAR, PATH_SEATBID_BID_ABC,
};
use crate::proto::com::iabtechlab::bidstream::mutation::v1::{RtbRequest, RtbResponse};

/// True when `intent` may be returned given the request's applicable intents.
/// An empty list means all intents are applicable, matching the Go
/// implementation's `IsIntentApplicable` semantics.
fn is_intent_applicable(intent: i32, applicable_intents: &[i32]) -> bool {
    applicable_intents.is_empty() || applicable_intents.contains(&intent)
}

/// Evaluate the `RtbRequest` and return an `RtbResponse`. Mutations are
/// limited to the intents the request declared applicable.
pub async fn evaluate(req: RtbRequest) -> RtbResponse {
    // For demonstration purposes, we will create a static response
    // In a real-world scenario, you would implement logic to evaluate the request
    // and determine the appropriate mutations based on the request data.
    let metadata = build_metadata();

    // Route on the auction id carried by the enclosed bid request; the
    // top-level `req.id` is the extension point envelope id assigned by the
    // exchange and is only echoed back in the response.
    let auction_id = req
        .bid_request
        .as_ref()
        .and_then(|bid_request| bid_request.id.clone())
        .unwrap_or_default();

    let applicable_intents = req.applicable_intents.clone();

    let mut response = match auction_id.as_str() {
        "auction-123" => RtbResponse {
            id: req.id,
            mutations: vec![
                activate_segments(&["seg-sports", "demo-25-35", "gender-male"]),
                activate_deals(PATH_IMP_1, &["display-deal-001"]),
            ],
            metadata: Some(metadata),
        },
        "auction-456" => RtbResponse {
            id: req.id,
            mutations: vec![
                activate_segments(&["demo-35-44"]),
                activate_deals(PATH_IMP_1, &["premium-deal-001", "video-deal-001"]),
            ],
            metadata: Some(metadata),
        },
        "auction-789" => RtbResponse {
            id: req.id,
            mutations: vec![
                activate_segments(&["demo-45-plus"]),
                activate_deals(PATH_IMP_1, &["display-deal-001"]),
                bid_shade(PATH_SEATBID_BID_ABC, 4.675),
            ],
            metadata: Some(metadata),
        },
        "auction-multi-123" => RtbResponse {
            id: req.id,
            mutations: vec![
                activate_segments(&["demo-35-44", "gender-female"]),
                activate_deals(PATH_IMP_HEADER, &["display-deal-001"]),
                activate_deals(PATH_IMP_SIDEBAR, &["display-deal-001"]),
                activate_deals(PATH_IMP_FOOTER, &["display-deal-001"]),
            ],
            metadata: Some(metadata),
        },
        "auction-native-123" => RtbResponse {
            id: req.id,
            mutations: vec![
                activate_segments(&["demo-18-24"]),
                activate_deals(PATH_IMP_1, &["native-deal-001"]),
            ],
            metadata: Some(metadata),
        },
        _ => RtbResponse {
            id: req.id,
            mutations: vec![],
            metadata: Some(metadata),
        },
    };

    response
        .mutations
        .retain(|mutation| is_intent_applicable(mutation.intent, &applicable_intents));

    response
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::mutation::types::{PATH_IMP_1, PATH_SEATBID_BID_ABC, PATH_USER_SEGMENT};
    use crate::proto::com::iabtechlab::bidstream::mutation::v1::{
        mutation::Value, Intent, Operation,
    };
    use crate::proto::com::iabtechlab::openrtb::v2::BidRequest;

    fn request(envelope_id: &str, auction_id: &str) -> RtbRequest {
        RtbRequest {
            id: envelope_id.to_string(),
            bid_request: Some(BidRequest {
                id: Some(auction_id.to_string()),
                ..Default::default()
            }),
            ..Default::default()
        }
    }

    #[tokio::test]
    async fn known_auctions_produce_expected_mutation_counts() {
        let cases = [
            ("auction-123", 2),
            ("auction-456", 2),
            ("auction-789", 3),
            ("auction-multi-123", 4),
            ("auction-native-123", 2),
        ];
        for (auction_id, expected) in cases {
            let response = evaluate(request("req-1", auction_id)).await;
            assert_eq!(response.mutations.len(), expected, "auction {auction_id}");
            assert!(response.metadata.is_some(), "auction {auction_id}");
        }
    }

    #[tokio::test]
    async fn response_echoes_envelope_id_not_auction_id() {
        let response = evaluate(request("envelope-42", "auction-123")).await;
        assert_eq!(response.id, "envelope-42");
    }

    #[tokio::test]
    async fn segment_and_deal_mutations_carry_expected_fields() {
        let response = evaluate(request("req-1", "auction-123")).await;

        let segments = &response.mutations[0];
        assert_eq!(segments.intent, Intent::ActivateSegments as i32);
        assert_eq!(segments.op, Operation::Add as i32);
        assert_eq!(segments.path, PATH_USER_SEGMENT);
        match segments.value.as_ref() {
            Some(Value::Ids(ids)) => {
                assert_eq!(ids.id, vec!["seg-sports", "demo-25-35", "gender-male"])
            }
            other => panic!("expected IDs payload, got {other:?}"),
        }

        let deals = &response.mutations[1];
        assert_eq!(deals.intent, Intent::ActivateDeals as i32);
        assert_eq!(deals.op, Operation::Add as i32);
        assert_eq!(deals.path, PATH_IMP_1);
    }

    #[tokio::test]
    async fn bid_shade_mutation_replaces_price_at_bid_path() {
        let response = evaluate(request("req-1", "auction-789")).await;

        let shade = response.mutations.last().expect("bid shade mutation");
        assert_eq!(shade.intent, Intent::BidShade as i32);
        assert_eq!(shade.op, Operation::Replace as i32);
        assert_eq!(shade.path, PATH_SEATBID_BID_ABC);
        match shade.value.as_ref() {
            Some(Value::AdjustBid(adjust)) => assert_eq!(adjust.price, 4.675),
            other => panic!("expected AdjustBid payload, got {other:?}"),
        }
    }

    #[tokio::test]
    async fn unknown_auction_returns_no_mutations() {
        let response = evaluate(request("req-1", "auction-unknown")).await;
        assert!(response.mutations.is_empty());
        assert!(response.metadata.is_some());
    }

    #[tokio::test]
    async fn mutations_are_limited_to_applicable_intents() {
        let mut req = request("req-1", "auction-789");
        req.applicable_intents = vec![Intent::BidShade as i32];

        let response = evaluate(req).await;

        assert_eq!(response.mutations.len(), 1);
        assert_eq!(response.mutations[0].intent, Intent::BidShade as i32);
    }

    #[tokio::test]
    async fn empty_applicable_intents_allows_all_mutations() {
        let response = evaluate(request("req-1", "auction-789")).await;
        assert_eq!(response.mutations.len(), 3);
    }

    #[tokio::test]
    async fn no_matching_applicable_intent_returns_no_mutations() {
        let mut req = request("req-1", "auction-123");
        req.applicable_intents = vec![Intent::AdjustDealFloor as i32];

        let response = evaluate(req).await;

        assert!(response.mutations.is_empty());
        assert!(response.metadata.is_some());
    }

    #[tokio::test]
    async fn missing_bid_request_returns_no_mutations() {
        let request = RtbRequest {
            id: "req-1".to_string(),
            ..Default::default()
        };
        let response = evaluate(request).await;
        assert_eq!(response.id, "req-1");
        assert!(response.mutations.is_empty());
    }
}
