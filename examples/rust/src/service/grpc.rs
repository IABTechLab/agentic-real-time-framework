//! gRPC service implementation for the RTB extension point.

use tonic::{Request, Response, Status};

use crate::bidder::evaluate;
use crate::proto::com::iabtechlab::bidstream::mutation::services::v1::rtb_extension_point_server;
use crate::proto::com::iabtechlab::bidstream::mutation::v1::{RtbRequest, RtbResponse};

/// gRPC service that dispatches requests to the bidder evaluator.
#[derive(Default)]
pub struct RtbExtensionPointService {}

#[tonic::async_trait]
impl rtb_extension_point_server::RtbExtensionPoint for RtbExtensionPointService {
    async fn get_mutations(
        &self,
        request: Request<RtbRequest>,
    ) -> Result<Response<RtbResponse>, Status> {
        let response: RtbResponse = evaluate(request.into_inner()).await;
        Ok(Response::new(response))
    }
}
