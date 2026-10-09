//! Environment-driven configuration for server startup.

use std::env;

/// Public API version for mutation responses.
pub const API_VERSION: &str = "1.0.0";
/// Model version used to generate mutations.
pub const MODEL_VERSION: &str = "v0.10.0";

/// Application configuration derived from environment variables.
#[derive(Clone, Debug)]
pub struct Config {
    pub address: String,
    pub grpc_port: u16,
    pub http_port: u16,
    pub max_server_connection: u16,
}

impl Config {
    /// Build configuration from environment variables with defaults.
    pub fn from_env() -> Self {
        let address = env::var("ARTF_GRPC_SERVER_ADDRESS").unwrap_or_else(|_| "0.0.0.0".to_string());
        let grpc_port = env::var("ARTF_GRPC_SERVER_PORT")
            .unwrap_or_else(|_| "50051".to_string())
            .parse::<u16>()
            .unwrap();
        let http_port = env::var("ARTF_HTTP_SERVER_PORT")
            .unwrap_or_else(|_| "8080".to_string())
            .parse::<u16>()
            .unwrap();
        let max_server_connection = env::var("ARTF_MAX_CONNS")
            .unwrap_or_else(|_| "256".to_string())
            .parse::<u16>()
            .unwrap();

        Self {
            address,
            grpc_port,
            http_port,
            max_server_connection,
        }
    }
}
