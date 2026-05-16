use async_trait::async_trait;

use crate::error::Result;
use crate::sample::Sample;

#[async_trait]
pub trait Collector: Send + Sync {
    fn name(&self) -> &'static str;

    async fn collect(&mut self) -> Result<Vec<Sample>>;
}
