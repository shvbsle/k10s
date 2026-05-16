use std::collections::BTreeMap;
use std::fmt::Write;
use std::net::SocketAddr;
use std::sync::Arc;

use async_trait::async_trait;
use axum::extract::State;
use axum::http::header;
use axum::response::IntoResponse;
use axum::routing::get;
use axum::Router;
use parking_lot::RwLock;
use tokio_util::sync::CancellationToken;
use tracing::info;

use crate::emitter::Emitter;
use crate::error::Result;
use crate::sample::{MetricValue, Sample};

type MetricKey = (String, BTreeMap<String, String>);
type MetricStore = Arc<RwLock<BTreeMap<MetricKey, f64>>>;

pub struct PrometheusEmitter {
    store: MetricStore,
}

impl PrometheusEmitter {
    pub fn new(port: u16, shutdown: CancellationToken) -> Self {
        let store: MetricStore = Arc::new(RwLock::new(BTreeMap::new()));
        let server_store = store.clone();

        tokio::spawn(async move {
            let app = Router::new()
                .route("/metrics", get(metrics_handler))
                .route("/health", get(|| async { "ok" }))
                .with_state(server_store);

            let addr = SocketAddr::from(([0, 0, 0, 0], port));
            let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
            info!(%addr, "prometheus endpoint listening");

            axum::serve(listener, app)
                .with_graceful_shutdown(shutdown.cancelled_owned())
                .await
                .unwrap();
        });

        Self { store }
    }
}

#[async_trait]
impl Emitter for PrometheusEmitter {
    fn name(&self) -> &'static str {
        "prometheus"
    }

    async fn emit(&mut self, samples: &[Sample]) -> Result<()> {
        let mut store = self.store.write();
        for sample in samples {
            let key = (sample.metric.clone(), sample.labels.clone());
            let value = match &sample.value {
                MetricValue::Gauge(v) => *v,
                MetricValue::Counter(v) => *v as f64,
            };
            store.insert(key, value);
        }
        Ok(())
    }
}

async fn metrics_handler(State(store): State<MetricStore>) -> impl IntoResponse {
    let store = store.read();
    let mut output = String::with_capacity(4096);

    for ((metric, labels), value) in store.iter() {
        let _ = write!(output, "{metric}");
        if !labels.is_empty() {
            output.push('{');
            for (i, (k, v)) in labels.iter().enumerate() {
                if i > 0 {
                    output.push(',');
                }
                let _ = write!(output, "{k}=\"{v}\"");
            }
            output.push('}');
        }
        let _ = writeln!(output, " {value}");
    }

    (
        [
            (
                header::CONTENT_TYPE,
                "text/plain; version=0.0.4; charset=utf-8",
            ),
            (header::ACCESS_CONTROL_ALLOW_ORIGIN, "*"),
        ],
        output,
    )
}
