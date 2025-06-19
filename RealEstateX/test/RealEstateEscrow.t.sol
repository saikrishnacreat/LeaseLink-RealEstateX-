// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.19;

import "forge-std/Test.sol";
import "../src/RealEstateEscrow.sol";
import {MockV3Aggregator} from "../test/Mocks/MockV3Aggregator.sol";

contract RealEstateEscrowTest is Test {
    RealEstateEscrow public escrow;
    MockV3Aggregator public mockPriceFeed;

    address landlord = address(0x1);
    address tenant = address(this); // this test contract will be the tenant

    function setUp() public {
        mockPriceFeed = new MockV3Aggregator(8, 3000e8); // 1 ETH = $3000
        escrow = new RealEstateEscrow(address(mockPriceFeed));
    }

   function testCreateAndWithdrawRental() public {
    uint256 rentUSD = 3000;
    uint256 duration = 1 days;
    uint256 rentETH = escrow.getETHAmountFromUSD(rentUSD);

    address payable mockLandlord = payable(address(0xBEEF));
    vm.deal(address(this), rentETH);

    escrow.createRental{value: rentETH}(mockLandlord, rentUSD, duration);

    vm.warp(block.timestamp + duration + 1);

    vm.prank(mockLandlord);
    escrow.withdrawRent(0);
}

}
