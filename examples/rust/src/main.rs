//! Application entry point and server orchestration.

use hyper::service::{make_service_fn, service_fn};
use hyper::Server as HyperServer;
use std::convert::Infallible;
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::signal;
use tokio::sync::watch;
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

    // Shared shutdown flag: set on SIGINT/SIGTERM, or when either server exits,
    // so the other server drains in-flight requests and stops too.
    let (shutdown_tx, shutdown_rx) = watch::channel(false);
    let shutdown_tx = Arc::new(shutdown_tx);

    let signal_tx = shutdown_tx.clone();
    tokio::spawn(async move {
        shutdown_signal().await;
        println!("Shutdown signal received, draining servers...");
        signal_tx.send_replace(true);
    });

    let max_server_connection = config.max_server_connection;
    let mut grpc_shutdown = shutdown_rx.clone();
    let grpc_tx = shutdown_tx.clone();
    let grpc_server = tokio::spawn(async move {
        let result = Server::builder()
            .concurrency_limit_per_connection(max_server_connection as usize)
            .add_service(agentic_rtb_framework_service)
            .serve_with_shutdown(grpc_addr, async move {
                let _ = grpc_shutdown.wait_for(|stop| *stop).await;
            })
            .await;
        grpc_tx.send_replace(true);
        report("gRPC", result.map_err(|e| e.to_string()))
    });

    let rest_address = address.clone();
    let http_port = config.http_port;
    let mut rest_shutdown = shutdown_rx;
    let rest_tx = shutdown_tx;
    let rest_server = tokio::spawn(async move {
        let make_svc = make_service_fn(|_conn| async {
            Ok::<_, Infallible>(service_fn(handle_rest))
        });

        let rest_addr: SocketAddr = format!("{}:{}", rest_address, http_port)
            .parse()
            .unwrap();

        let result = match HyperServer::try_bind(&rest_addr) {
            Ok(builder) => {
                builder
                    .serve(make_svc)
                    .with_graceful_shutdown(async move {
                        let _ = rest_shutdown.wait_for(|stop| *stop).await;
                    })
                    .await
            }
            Err(e) => Err(e),
        };
        rest_tx.send_replace(true);
        report("HTTP", result.map_err(|e| e.to_string()))
    });

    let (grpc_result, rest_result) = tokio::join!(grpc_server, rest_server);
    let mut healthy = true;
    for (name, result) in [("gRPC", grpc_result), ("HTTP", rest_result)] {
        match result {
            Ok(ok) => healthy &= ok,
            Err(e) => {
                eprintln!("{} Server task failed: {:?}", name, e);
                healthy = false;
            }
        }
    }

    if healthy {
        Ok(())
    } else {
        Err("one or more servers exited with an error".into())
    }
}

/// Resolve when the process receives SIGINT (Ctrl+C) or SIGTERM.
async fn shutdown_signal() {
    let ctrl_c = async {
        signal::ctrl_c().await.expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to install SIGTERM handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }
}

/// Log how a server finished and return whether it stopped cleanly.
fn report(name: &str, result: Result<(), String>) -> bool {
    match result {
        Ok(()) => {
            println!("{} Server stopped successfully.", name);
            true
        }
        Err(e) => {
            eprintln!("{} Server encountered an error: {}", name, e);
            false
        }
    }
}
