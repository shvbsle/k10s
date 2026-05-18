use async_trait::async_trait;

use crate::error::Result;
use crate::sample::Sample;

#[async_trait]
pub trait Emitter: Send + Sync {
    fn name(&self) -> &'static str;

    async fn emit(&mut self, samples: &[Sample]) -> Result<()>;
}
