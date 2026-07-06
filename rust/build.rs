use std::env;
use std::path::PathBuf;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let descriptor_path = PathBuf::from(env::var("OUT_DIR")?).join("artf_descriptor.bin");
    tonic_prost_build::configure()
        .build_server(true)
        .build_client(false)
        .out_dir("./src")
        .file_descriptor_set_path(descriptor_path)
        .compile_protos(&["./proto/agenticrtbframeworkservices.proto"], &["proto"])?;
    Ok(())
}
