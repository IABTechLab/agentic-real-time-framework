//! Application entry point and server orchestration.

use hyper::service::{make_service_fn, service_fn};
use hyper::Server as HyperServer;
use std::convert::Infallible;
use std::net::SocketAddr;
use tonic::transport::Server;

mod bidder;
mod config;
mod mutation;
mod proto;
mod service;

use crate::config::{Config, API_VERSION, MODEL_VERSION};
use crate::proto::com::iabtechlab::bidstream::mutation::services::v1::rtb_extension_point_server::RtbExtensionPointServer;
use crate::service::{handle_rest, RtbExtensionPointService};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let config = Config::from_env();
    let address = config.address.clone();

    let grpc_addr = format!("{}:{}", address.as_str(), config.grpc_port)
        .parse()
        .unwrap();
    let agentic_rtb_framework_service = RtbExtensionPointServer::new(RtbExtensionPointService::default());

    println!("Agentic RTB Framework API Version: {}", API_VERSION);
    println!("Agentic RTB Framework Model Version: {}", MODEL_VERSION);
    println!(
        "Setting gRPC Server Max connections: {}",
        config.max_server_connection
    );
    println!(
        "Starting gRPC Server at: {}:{}",
        address, config.grpc_port
    );
    println!(
        "Starting HTTP Server at: {}:{}",
        address, config.http_port
    );

    let max_server_connection = config.max_server_connection;
    let grpc_server = tokio::spawn(async move {
        Server::builder()
            .concurrency_limit_per_connection(max_server_connection as usize)
            .add_service(agentic_rtb_framework_service)
            .serve(grpc_addr)
            .await

    });

    let rest_address = address.clone();
    let http_port = config.http_port;
    let rest_server = tokio::spawn(async move {
        let make_svc = make_service_fn(|_conn| async {
            Ok::<_, Infallible>(service_fn(handle_rest))
        });

        let rest_addr: SocketAddr = format!("{}:{}", rest_address, http_port)
            .parse()
            .unwrap();

        HyperServer::bind(&rest_addr)
            .serve(make_svc)
            .await
    });

    // Wait for both servers (or handle their errors)
    tokio::select! {
        result = grpc_server => {
            match result {
                Ok(_) => println!("gRPC Server stopped successfully."),
                Err(e) => eprintln!("gRPC Server encountered an error: {:?}", e),
            }
        }
        result = rest_server => {
            match result {
                Ok(_) => println!("HTTP Server stopped successfully."),
                Err(e) => eprintln!("HTTP Server encountered an error: {:?}", e),
            }
        }
    }

    Ok(())
}
