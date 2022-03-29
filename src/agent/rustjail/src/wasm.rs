// Copyright (c) 2022 Ant Group
//
// SPDX-License-Identifier: Apache-2.0
//

use anyhow::Result;
use wasmer::{Instance, Module, Store};
use wasmer_compiler_cranelift::Cranelift;
use wasmer_engine_universal::Universal;
use wasmer_wasi::WasiState;

use crate::container::{CLOG_FD, CWFD_FD};
use crate::log_child;
use crate::sync::{write_count, write_sync, SYNC_FAILED};

pub fn run_wasm(rootfs: String, wasm_path: String, wargs: Vec<String>) {
    let cwfd = std::env::var(CWFD_FD).unwrap().parse::<i32>().unwrap();
    let cfd_log = std::env::var(CLOG_FD).unwrap().parse::<i32>().unwrap();

    log_child!(
        cfd_log,
        "run_wasm rootfs:{} wasm_path:{} wargs:{:?}",
        rootfs,
        wasm_path,
        wargs
    );

    match do_run_wasm(rootfs, wasm_path, wargs) {
        Ok(_) => log_child!(cfd_log, "run_wasm process exit successfully"),
        Err(e) => {
            log_child!(cfd_log, "run_wasm process exit:child exit: {:?}", e);
            let _ = write_sync(cwfd, SYNC_FAILED, format!("{:?}", e).as_str());
        }
    }
}

fn do_run_wasm(_rootfs: String, wasm_path: String, wargs: Vec<String>) -> Result<()> {
    let compiler_config = Cranelift::default();
    let engine = Universal::new(compiler_config).engine();
    let store = Store::new(&engine);

    let mut wasi_env = WasiState::new(wasm_path.clone()).args(wargs).finalize()?;

    let module = Module::from_file(&store, wasm_path)?;
    let import_object = wasi_env.import_object(&module)?;
    let instance = Instance::new(&module, &import_object)?;

    let start = instance.exports.get_function("_start")?;
    start.call(&[])?;

    Ok(())
}
