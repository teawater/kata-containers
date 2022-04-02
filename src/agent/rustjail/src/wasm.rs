// Copyright (c) 2022 Ant Group
//
// SPDX-License-Identifier: Apache-2.0
//

use anyhow::Result;
use std::os::unix::io::RawFd;
use std::process::exit;

use wasmer::{Instance, Module, Store};
use wasmer_compiler_cranelift::Cranelift;
use wasmer_engine_universal::Universal;
use wasmer_wasi::WasiState;

use crate::container::{
    do_child_setup, do_child_setup_release, CLOG_FD, CRFD_FD, CWFD_FD, FIFO_FD, INIT, NO_PIVOT,
};
use crate::log_child;
use crate::sync::{write_count, write_sync, SYNC_FAILED};

pub fn run_wasm() {
    let instance = do_setup_wrapper();
    let start = instance.exports.get_function("_start").unwrap();
    start.call(&[]).unwrap();
}

pub fn do_setup_wrapper() -> Instance {
    let cwfd = std::env::var(CWFD_FD).unwrap().parse::<i32>().unwrap();
    let cfd_log = std::env::var(CLOG_FD).unwrap().parse::<i32>().unwrap();

    match do_setup(cwfd, cfd_log) {
        Ok(instance) => {
            log_child!(cfd_log, "wasm do_setup successfully");
            instance
        }
        Err(e) => {
            log_child!(cfd_log, "wasm do_setup error {:?}", e);
            let _ = write_sync(cwfd, SYNC_FAILED, format!("{:?}", e).as_str());
            exit(0);
        }
    }
}

fn do_setup(cwfd: RawFd, cfd_log: RawFd) -> Result<Instance> {
    let init = std::env::var(INIT)?.eq(format!("{}", true).as_str());
    let no_pivot = std::env::var(NO_PIVOT)?.eq(format!("{}", true).as_str());
    let crfd = std::env::var(CRFD_FD)?.parse::<i32>().unwrap();

    let mut fifofd = -1;
    if init {
        fifofd = std::env::var(FIFO_FD)?.parse::<i32>().unwrap();
    }

    let (args, oci_process) = do_child_setup(cwfd, crfd, cfd_log, init, no_pivot)?;

    let instance = do_setup_wasm(args[0].clone(), &args[1..])?;

    log_child!(cfd_log, "ready to run wasm");

    do_child_setup_release(cwfd, crfd, cfd_log, fifofd, init, oci_process)?;

    Ok(instance)
}

fn do_setup_wasm(wasm_path: String, wargs: &[String]) -> Result<Instance> {
    let compiler_config = Cranelift::default();
    let engine = Universal::new(compiler_config).engine();
    let store = Store::new(&engine);

    let mut wasi_env = WasiState::new(wasm_path.clone()).args(wargs).finalize()?;

    let module = Module::from_file(&store, wasm_path)?;
    let import_object = wasi_env.import_object(&module)?;
    let instance = Instance::new(&module, &import_object)?;

    let _ = instance.exports.get_function("_start")?;

    Ok(instance)
}
