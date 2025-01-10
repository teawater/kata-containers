//
// Copyright (c) 2025 Ant Group
//
// SPDX-License-Identifier: Apache-2.0
//

use anyhow::{Context, Result};
use async_trait::async_trait;
use protocols::empty;
use protocols::{attestation_agent, attestation_agent_ttrpc};
use rustjail::container::{BaseContainer, LinuxContainer};
use slog::{debug, error};
use std::sync::Arc;
use tokio::sync::mpsc;
use tokio::sync::Mutex;
use ttrpc::r#async::Server as TtrpcServer;

const SOCK_ADDR_DIR: &str = "/run/kata-containers/";
const SOCK_ADDR: &str = r"unix:///run/kata-containers/attestation_agent.sock";

fn sl() -> slog::Logger {
    slog_scope::logger().new(o!("subsystem" => "attestation_agent"))
}

fn remove_if_sock_exist(sock_addr: &str) -> Result<()> {
    let path = sock_addr
        .strip_prefix("unix://")
        .expect("socket address is not expected");

    if std::path::Path::new(path).exists() {
        std::fs::remove_file(path)?;
    }

    Ok(())
}

struct AttestationAgentService {
    event_rx: Arc<Mutex<mpsc::Receiver<attestation_agent::Event>>>,
    fail_event: Arc<Mutex<Option<attestation_agent::Event>>>,
}

#[async_trait]
impl attestation_agent_ttrpc::AttestationAgent for AttestationAgentService {
    async fn events_stream(
        &self,
        _ctx: &::ttrpc::r#async::TtrpcContext,
        _: empty::Empty,
        s: ::ttrpc::r#async::ServerStreamSender<attestation_agent::Event>,
    ) -> ::ttrpc::Result<()> {
        info!(sl(), "container_events_stream connected");

        let oevent = self.fail_event.lock().await.take();
        if let Some(event) = oevent {
            debug!(sl(), "fail event {:?}", event);
            if let Err(e) = s.send(&event).await {
                error!(sl(), "send fail event {:?} failed: {:?}", event, e);
                let _ = self.fail_event.lock().await.insert(event);
                return Err(e.into());
            } else {
                debug!(sl(), "send fail event {:?} success", event);
            }
        }

        while let Some(event) = self.event_rx.lock().await.recv().await {
            debug!(sl(), "new event {:?}", event);
            if let Err(e) = s.send(&event).await {
                error!(sl(), "send event {:?} failed: {:?}", event, e);
                let _ = self.fail_event.lock().await.insert(event);
                break;
            } else {
                debug!(sl(), "send event {:?} success", event);
            }
        }

        Ok(())
    }
}

pub struct AttestationAgent {
    _server: ttrpc::asynchronous::Server,
    event_tx: mpsc::Sender<attestation_agent::Event>,
}

impl std::fmt::Debug for AttestationAgent {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("AttestationAgent").finish()
    }
}

impl AttestationAgent {
    pub async fn new() -> Result<Self> {
        let (tx, rx) = mpsc::channel(128);

        let s = Box::new(AttestationAgentService {
            event_rx: Arc::new(Mutex::new(rx)),
            fail_event: Arc::new(Mutex::new(None)),
        }) as Box<dyn attestation_agent_ttrpc::AttestationAgent + Send + Sync>;
        let s = Arc::new(s);
        let service = attestation_agent_ttrpc::create_attestation_agent(s);
        remove_if_sock_exist(SOCK_ADDR)?;
        std::fs::create_dir_all(SOCK_ADDR_DIR)?;

        let mut server = TtrpcServer::new()
            .bind(SOCK_ADDR)?
            .register_service(service);

        server
            .start()
            .await
            .context("failed to start AttestationAgent ttrpc server")?;

        info!(sl(), "server started");

        Ok(Self {
            _server: server,
            event_tx: tx,
        })
    }

    async fn send_event(&self, event: attestation_agent::Event) -> Result<()> {
        self.event_tx
            .try_send(event.clone())
            .context(format!("failed to send event {:?} to channel", event))?;

        debug!(sl(), "sent event {:?} to channel", event);

        Ok(())
    }

    pub async fn send_container_event(
        &self,
        c: &LinuxContainer,
        action: &str,
    ) -> Result<()> {
        let ci = c.to_attestation_agent_container_info()?;

        let mut event = attestation_agent::Event::new();
        event.set_type("container".to_string());
        event.set_action(action.to_string());
        event.set_container(ci);

        self.send_event(event).await?;

        Ok(())
    }
}
