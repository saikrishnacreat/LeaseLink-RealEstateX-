
# 🏠 RealEstate Rental DApp

A decentralized platform enabling tenants to pay rent in **ETH** (pegged to USD) using **Chainlink Price Feeds**. Landlords can securely withdraw rent after the rental period ends.

---

## 🔗 Live Demo

> [Coming Soon – Add your deployed URL here]

---

## 📜 Smart Contract Features

- Tenants create rental agreements by paying ETH equivalent to a specified USD amount.
- Real-time ETH/USD conversion via Chainlink Price Feeds.
- Landlords withdraw rent after the rental duration.
- Escrow-like mechanism: funds are locked until rental completion.

---

## 💡 Technologies Used

| Layer             | Tech                                      |
|-------------------|-------------------------------------------|
| **Frontend**      | React.js, Wagmi, Viem (ethers optional)   |
| **Backend**       | Optional: Express.js                      |
| **Smart Contracts** | Solidity, Foundry or Hardhat            |
| **Oracles**       | Chainlink ETH/USD Price Feed              |
| **Network**       | Sepolia Testnet                           |

---

## ⚙️ Project Structure

```
realestate-rental/
├── contracts/
│   └── RealEstateRental.sol      # Main smart contract
├── frontend/
│   └── src/
│       ├── App.jsx              # React UI
│       ├── components/          # Rental form, history, etc.
│       └── contracts/           # ABI & deployed addresses
├── scripts/                     # Deployment scripts (Foundry/Hardhat)
├── foundry.toml / hardhat.config.js
└── README.md
```

---

## 📦 Smart Contract: `RealEstateRental.sol`

### 🔐 Rental Struct

```solidity
struct Rental {
    address tenant;
    address payable landlord;
    uint256 rentUSD;
    uint256 rentETH;
    uint256 startTime;
    uint256 duration;
    bool isActive;
    bool isWithdrawn;
}
```

### 🛠️ Key Functions

```solidity
function createRental(address landlord, uint256 rentUSD, uint256 duration) external payable;
function withdrawRent(uint256 rentalId) external;
function getLatestPrice() public view returns (int);
function getAllRentals() public view returns (Rental[] memory);
```

---

## 🖥️ Frontend Features

- Wallet connect (MetaMask)
- Create Rental Form (inputs: landlord, USD rent, duration)
- Rental History (tenant/landlord info, createdAt date, status)
- Withdraw button for landlords (enabled after duration)

---

## 🚀 Getting Started

### 🔧 Deploy Contract

Using Foundry:

```bash
forge create --rpc-url <SEPOLIA_RPC> --private-key <PRIVATE_KEY> src/RealEstateRental.sol:RealEstateRental
```

Or using Hardhat:

```bash
npx hardhat run scripts/deploy.js --network sepolia
```

### 🌐 Run Frontend

```bash
cd frontend
npm install
npm run dev
```

### 🧪 Testing

With Foundry:

```bash
forge test
```

Or with Hardhat:

```bash
npx hardhat test
```

---

## ✨ Example Rental History Display

```
Tenant:    0x82Ff...
Landlord:  0x1234...
Rent:      50 USD
Duration:  604800 seconds (1 week)
Created:   2025-06-19 10:24 AM
```

---

## 🙋‍♂️ Author

- **Saikrishna A.**
- [GitHub](#) <!-- Add your GitHub profile link -->
- ✉️ saikrishnaask191@gmail..com

---

## 📄 License

MIT License – feel free to use and build upon this project.
