//! Bid evaluation logic that generates mutation responses.

use crate::mutation::builder::{activate_deals, activate_segments, bid_shade, build_metadata};
use crate::mutation::types::{
    PATH_IMP_1, PATH_IMP_FOOTER, PATH_IMP_HEADER, PATH_IMP_SIDEBAR, PATH_SEATBID_BID_ABC,
};
use crate::proto::com::iabtechlab::bidstream::mutation::v1::{RtbRequest, RtbResponse};

/// Evaluate the `RtbRequest` and return an `RtbResponse`.
pub async fn evaluate(req: RtbRequest) -> RtbResponse {
    // For demonstration purposes, we will create a static response
    // In a real-world scenario, you would implement logic to evaluate the request
    // and determine the appropriate mutations based on the request data.
    let metadata = build_metadata();

    match req.id.as_str() {
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
        "app-123" => RtbResponse {
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
    }
}
