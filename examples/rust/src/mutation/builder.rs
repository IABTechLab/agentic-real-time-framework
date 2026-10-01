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
