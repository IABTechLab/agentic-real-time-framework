//! Service-layer modules for gRPC and HTTP endpoints.

pub mod grpc;
pub mod http;

pub use grpc::RtbExtensionPointService;
pub use http::handle_rest;
