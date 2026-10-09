//! HTTP handlers for health and readiness endpoints.

use hyper::body::Body;
use hyper::{Method, StatusCode};
use std::convert::Infallible;

/// Handle simple health and readiness routes.
pub async fn handle_rest(
    req: hyper::Request<Body>,
) -> Result<hyper::Response<Body>, Infallible> {
    match (req.method(), req.uri().path()) {
        (&Method::GET, "/health/live") => Ok(hyper::Response::new(Body::from("OK"))),
        (&Method::GET, "/health/ready") => Ok(hyper::Response::new(Body::from("OK"))),
        _ => Ok(hyper::Response::builder()
            .status(StatusCode::NOT_FOUND)
            .body(Body::from("Not Found"))
            .unwrap()),
    }
}
