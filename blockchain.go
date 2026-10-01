package cryptures

// BlockchainService groups the blockchain-domain operations: chain data and
// market data, transaction building and broadcasting, HD wallets, token
// contracts, fee estimation, chain lookups, NFTs, and IPFS storage.
type BlockchainService struct {
	// Data covers balances, transaction history, portfolios, exchange rates,
	// address screening, and market data.
	Data *BlockchainDataService
	// Operations covers building, signing, and broadcasting transactions, and
	// the raw JSON-RPC gateway.
	Operations *BlockchainOperationsService
	// Wallet covers HD wallet generation and address/private-key derivation.
	Wallet *BlockchainWalletService
	// Contracts covers fungible token contract deployment, minting, and
	// burning.
	Contracts *BlockchainContractsService
	// Fee covers network fee tiers and EVM gas estimation.
	Fee *BlockchainFeeService
	// Lookups covers transactions by hash, blocks, token metadata, and UTXOs.
	Lookups *BlockchainLookupsService
	// NFT covers NFT collection listings and token ownership.
	NFT *BlockchainNFTService
	// Storage covers IPFS uploads.
	Storage *BlockchainStorageService
}

// TxIDResponse is returned by every operation that signs and broadcasts a
// transaction.
type TxIDResponse struct {
	// TxID is the broadcast transaction hash/id.
	TxID string `json:"txId"`
}
