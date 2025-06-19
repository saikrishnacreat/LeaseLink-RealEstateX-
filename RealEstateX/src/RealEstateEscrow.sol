// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@chainlink/contracts/src/v0.8/shared/interfaces/AggregatorV3Interface.sol";

contract RealEstateEscrow {
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

    AggregatorV3Interface public priceFeed;
    uint256 public rentalIdCounter;
    mapping(uint256 => Rental) public rentals;

    event RentalCreated(uint256 rentalId, address tenant, address landlord, uint256 rentUSD, uint256 duration);
    event RentDeposited(uint256 rentalId, uint256 amount);
    event RentWithdrawn(uint256 rentalId);

    constructor(address _priceFeed) {
        priceFeed = AggregatorV3Interface(_priceFeed);
    }

    function createRental(address payable _landlord, uint256 _rentUSD, uint256 _durationInSeconds) external payable {
        uint256 rentETH = getETHAmountFromUSD(_rentUSD);
        require(msg.value >= rentETH, "Insufficient ETH");

        rentals[rentalIdCounter] = Rental({
            tenant: msg.sender,
            landlord: _landlord,
            rentUSD: _rentUSD,
            rentETH: rentETH,
            startTime: block.timestamp,
            duration: _durationInSeconds,
            isActive: true,
            isWithdrawn: false
        });

        emit RentalCreated(rentalIdCounter, msg.sender, _landlord, _rentUSD, _durationInSeconds);
        emit RentDeposited(rentalIdCounter, msg.value);

        rentalIdCounter++;
    }

    function withdrawRent(uint256 _rentalId) external {
        Rental storage rental = rentals[_rentalId];
        require(msg.sender == rental.landlord, "Only landlord");
        require(rental.isActive && !rental.isWithdrawn, "Already withdrawn or inactive");
        require(block.timestamp >= rental.startTime + rental.duration, "Rental period not over");

        rental.isWithdrawn = true;
        rental.isActive = false;
        rental.landlord.transfer(rental.rentETH);

        emit RentWithdrawn(_rentalId);
    }

    function getLatestPrice() public view returns (int256) {
        (, int256 price,,,) = priceFeed.latestRoundData();
        return price; // 8 decimals
    }

    function getETHAmountFromUSD(uint256 usdAmount) public view returns (uint256) {
        int256 price = getLatestPrice();
        require(price > 0, "Invalid price");
        uint256 decimals = priceFeed.decimals();
        return (usdAmount * (10 ** 18)) / uint256(price) * (10 ** decimals) / (10 ** decimals);
    }

    // ✅ Return all rentals (for admin/frontend display)
    function getAllRentals() external view returns (Rental[] memory) {
        Rental[] memory all = new Rental[](rentalIdCounter);
        for (uint256 i = 0; i < rentalIdCounter; i++) {
            all[i] = rentals[i];
        }
        return all;
    }

    // ✅ Return only the sender's rentals (tenant or landlord)
    function getMyRentals() external view returns (Rental[] memory) {
        uint256 count = 0;
        for (uint256 i = 0; i < rentalIdCounter; i++) {
            if (rentals[i].tenant == msg.sender || rentals[i].landlord == msg.sender) {
                count++;
            }
        }

        Rental[] memory result = new Rental[](count);
        uint256 index = 0;
        for (uint256 i = 0; i < rentalIdCounter; i++) {
            if (rentals[i].tenant == msg.sender || rentals[i].landlord == msg.sender) {
                result[index++] = rentals[i];
            }
        }

        return result;
    }
}
