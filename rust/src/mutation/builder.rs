//! Builders for common mutation shapes.

use crate::config::{API_VERSION, MODEL_VERSION};
use crate::mutation::types::PATH_USER_SEGMENT;
use crate::proto::com::iabtechlab::bidstream::mutation::v1::{
    mutation::Value, AdjustBidPayload, IDsPayload, Intent, Metadata, Mutation, Operation,
};

/// Build response metadata from configured versions.
pub fn build_metadata() -> Metadata {
    Metadata {
        api_version: API_VERSION.to_string(),
        model_version: MODEL_VERSION.to_string(),
    }
}

/// Build a mutation to activate user segments.
pub fn activate_segments(ids: &[&str]) -> Mutation {
    Mutation {
        intent: Intent::ActivateSegments.into(),
        op: Operation::Add.into(),
        path: PATH_USER_SEGMENT.to_string(),
        value: Some(Value::Ids(ids_payload(ids))),
    }
}

/// Build a mutation to activate deals for the given impression path.
pub fn activate_deals(path: &str, ids: &[&str]) -> Mutation {
    Mutation {
        intent: Intent::ActivateDeals.into(),
        op: Operation::Add.into(),
        path: path.to_string(),
        value: Some(Value::Ids(ids_payload(ids))),
    }
}

/// Build a mutation that adjusts bid price at a bid path.
pub fn bid_shade(path: &str, price: f64) -> Mutation {
    Mutation {
        intent: Intent::BidShade.into(),
        op: Operation::Replace.into(),
        path: path.to_string(),
        value: Some(Value::AdjustBid(AdjustBidPayload { price: price })),
    }
}

fn ids_payload(ids: &[&str]) -> IDsPayload {
    IDsPayload {
        id: ids.iter().map(|id| id.to_string()).collect(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn metadata_uses_configured_versions() {
        let metadata = build_metadata();
        assert_eq!(metadata.api_version, API_VERSION);
        assert_eq!(metadata.model_version, MODEL_VERSION);
    }

    #[test]
    fn activate_segments_adds_ids_at_user_segment_path() {
        let mutation = activate_segments(&["seg-1", "seg-2"]);
        assert_eq!(mutation.intent, Intent::ActivateSegments as i32);
        assert_eq!(mutation.op, Operation::Add as i32);
        assert_eq!(mutation.path, PATH_USER_SEGMENT);
        match mutation.value {
            Some(Value::Ids(ids)) => assert_eq!(ids.id, vec!["seg-1", "seg-2"]),
            other => panic!("expected IDs payload, got {other:?}"),
        }
    }

    #[test]
    fn activate_deals_adds_ids_at_given_path() {
        let mutation = activate_deals("/imp/imp-1", &["deal-1"]);
        assert_eq!(mutation.intent, Intent::ActivateDeals as i32);
        assert_eq!(mutation.op, Operation::Add as i32);
        assert_eq!(mutation.path, "/imp/imp-1");
        match mutation.value {
            Some(Value::Ids(ids)) => assert_eq!(ids.id, vec!["deal-1"]),
            other => panic!("expected IDs payload, got {other:?}"),
        }
    }

    #[test]
    fn bid_shade_replaces_price_at_given_path() {
        let mutation = bid_shade("/seatbid/dsp-001/bid/bid-abc", 4.675);
        assert_eq!(mutation.intent, Intent::BidShade as i32);
        assert_eq!(mutation.op, Operation::Replace as i32);
        assert_eq!(mutation.path, "/seatbid/dsp-001/bid/bid-abc");
        match mutation.value {
            Some(Value::AdjustBid(adjust)) => assert_eq!(adjust.price, 4.675),
            other => panic!("expected AdjustBid payload, got {other:?}"),
        }
    }
}
