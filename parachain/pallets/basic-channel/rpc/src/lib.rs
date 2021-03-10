use codec::Codec;
use jsonrpc_core::{Result, Error as JsonError};
use jsonrpc_derive::rpc;
use sp_core::H256;
use sp_runtime::offchain::storage::StorageValueRef;

use artemis_basic_channel::outbound::{CommitmentData, generate_merkle_proofs, offchain_key};

type Proofs = Vec<Vec<u8>>;

#[rpc]
pub trait BasicChannelApi
{
	#[rpc(name = "get_merkle_proofs")]
	fn get_merkle_proofs(&self, root: H256) -> Result<Proofs>;
}

pub struct BasicChannel<AccountId> {
	indexing_prefix: &'static [u8],
	_marker: std::marker::PhantomData<AccountId>
}

impl<AccountId> BasicChannel<AccountId> {
	pub fn new(indexing_prefix: &'static [u8]) -> Self {
		Self {
			indexing_prefix,
			_marker: Default::default(),
		}
	}
}

impl<AccountId> BasicChannelApi for BasicChannel<AccountId>
where
	AccountId: Codec + Send + Sync + 'static,
{
	fn get_merkle_proofs(&self, root: H256) -> Result<Proofs> {
		let key = offchain_key(self.indexing_prefix, root);
		let data = StorageValueRef::persistent(&key);

		if let Some(Some(cdata)) = data.get::<CommitmentData<AccountId>>() {
			let subcommitments: Vec<Vec<u8>> = cdata.subcommitments.into_iter().map(|c| c.1).collect();
			let proofs = generate_merkle_proofs(subcommitments.into_iter());
			match proofs {
				Ok(proofs) => Ok(proofs),
				Err(_) => Err(JsonError::invalid_request()),
			}
		} else {
			Err(JsonError::invalid_request())
                }
	}
}
