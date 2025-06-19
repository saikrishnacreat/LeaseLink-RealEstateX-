RealEstateX/
├── src/
│   ├── RealEstateEscrow.sol         # Main escrow contract
│   ├── EscrowTypes.sol              # Structs/enums (optional)
│   └── utils/
│       └── PriceConverter.sol       # Chainlink price feed utility
│
├── script/
│   ├── DeployEscrow.s.sol           # Script to deploy contract
│   ├── CallFunctions.s.sol          # (Optional) Call Chainlink Functions manually
│   └── Utils.s.sol                  # Helper scripts
│
├── test/
│   ├── RealEstateEscrow.t.sol       # Main unit test file
│   └── Mocks/
│       └── MockV3Aggregator.sol     # Mock Chainlink Price Feed
│
├── chainlink/
│   └── functions-request-config.js  # Chainlink Functions config (JS CLI-based)
│
├── foundry.toml                     # Foundry config
├── .env                             # Store API keys, private key, etc.
├── README.md
└── lib/
    └── chainlink-brownie-contracts/ # Installed using forge install (for mocks)
