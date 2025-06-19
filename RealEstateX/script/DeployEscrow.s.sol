// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "forge-std/Script.sol";
import "../src/RealEstateEscrow.sol";

contract DeployEscrowScript is Script {
    function run() external {
        vm.startBroadcast();

        // Use Sepolia ETH/USD Price Feed: 0x694AA1769357215DE4FAC081bf1f309aDC325306
        address priceFeed = 0x694AA1769357215DE4FAC081bf1f309aDC325306;
        new RealEstateEscrow(priceFeed);

        vm.stopBroadcast();
    }
}
