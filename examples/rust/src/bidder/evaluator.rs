//! Bid evaluation logic that generates mutation responses.

use crate::mutation::builder::{activate_deals, activate_segments, bid_shade, build_metadata};
use crate::mutation::types::{
    PATH_IMP_1, PATH_IMP_FOOTER, PATH_IMP_HEADER, PATH_IMP_SIDEBAR, PATH_SEATBID_BID_ABC,
};
use crate::proto::com::iabtechlab::bidstream::mutation::v1::{Mutation, RtbRequest, RtbResponse};

/// Evaluate the `RtbRequest` and return an `RtbResponse`.
pub async fn evaluate(req: RtbRequest) -> RtbResponse {
    // For demonstration purposes, we will create a static response
    // In a real-world scenario, you would implement logic to evaluate the request
    // and determine the appropriate mutations based on the request data.
    let mut mutations: Vec<Mutation> = match req.id.as_str() {
        "auction-123" => vec![
            activate_segments(&["seg-sports", "demo-25-35", "gender-male"]),
            activate_deals(PATH_IMP_1, &["display-deal-001"]),
        ],
        "auction-456" => vec![
            activate_segments(&["demo-35-44"]),
            activate_deals(PATH_IMP_1, &["premium-deal-001", "video-deal-001"]),
        ],
        "auction-789" => vec![
            activate_segments(&["demo-45-plus"]),
            activate_deals(PATH_IMP_1, &["display-deal-001"]),
            bid_shade(PATH_SEATBID_BID_ABC, 4.675),
        ],
        "auction-multi-123" => vec![
            activate_segments(&["demo-35-44", "gender-female"]),
            activate_deals(PATH_IMP_HEADER, &["display-deal-001"]),
            activate_deals(PATH_IMP_SIDEBAR, &["display-deal-001"]),
            activate_deals(PATH_IMP_FOOTER, &["display-deal-001"]),
        ],
        "app-123" => vec![
            activate_segments(&["demo-18-24"]),
            activate_deals(PATH_IMP_1, &["native-deal-001"]),
        ],
        _ => vec![],
    };

    // Only return mutations whose intent the host declared as applicable.
    mutations.retain(|m| is_intent_applicable(m.intent, &req.applicable_intents));

    RtbResponse {
        id: req.id,
        mutations,
        metadata: Some(build_metadata()),
    }
}

/// An empty `applicable_intents` list means every intent is applicable.
fn is_intent_applicable(intent: i32, applicable_intents: &[i32]) -> bool {
    applicable_intents.is_empty() || applicable_intents.contains(&intent)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::proto::com::iabtechlab::bidstream::mutation::v1::Intent;

    fn request(id: &str, applicable_intents: &[Intent]) -> RtbRequest {
        RtbRequest {
            id: id.to_string(),
            applicable_intents: applicable_intents.iter().map(|i| *i as i32).collect(),
            ..Default::default()
        }
    }

    fn intents(resp: &RtbResponse) -> Vec<i32> {
        resp.mutations.iter().map(|m| m.intent).collect()
    }

    #[tokio::test]
    async fn returns_all_mutations_when_no_intents_specified() {
        let resp = evaluate(request("auction-789", &[])).await;
        assert_eq!(resp.id, "auction-789");
        assert_eq!(
            intents(&resp),
            vec![
                Intent::ActivateSegments as i32,
                Intent::ActivateDeals as i32,
                Intent::BidShade as i32,
            ]
        );
        assert!(resp.metadata.is_some());
    }

    #[tokio::test]
    async fn filters_mutations_by_applicable_intents() {
        let resp = evaluate(request("auction-789", &[Intent::ActivateSegments])).await;
        assert_eq!(intents(&resp), vec![Intent::ActivateSegments as i32]);

        let resp = evaluate(request(
            "auction-789",
            &[Intent::ActivateDeals, Intent::BidShade],
        ))
        .await;
        assert_eq!(
            intents(&resp),
            vec![Intent::ActivateDeals as i32, Intent::BidShade as i32]
        );
    }

    #[tokio::test]
    async fn returns_no_mutations_when_no_intent_matches() {
        let resp = evaluate(request("auction-123", &[Intent::AddMetrics])).await;
        assert!(resp.mutations.is_empty());
    }

    #[tokio::test]
    async fn unknown_request_returns_empty_response() {
        let resp = evaluate(request("unknown", &[])).await;
        assert_eq!(resp.id, "unknown");
        assert!(resp.mutations.is_empty());
        assert!(resp.metadata.is_some());
    }
}
