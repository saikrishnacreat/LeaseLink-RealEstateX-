import { useState, useEffect } from 'react';
import { useWriteContract, useWaitForTransactionReceipt, usePublicClient } from 'wagmi';
import { CONTRACT_ABI, CONTRACT_ADDRESS } from '../utils/constants';
import { formatEther } from 'viem';

export default function CreateRentalForm() {
  const [landlord, setLandlord] = useState('');
  const [rentUSD, setRentUSD] = useState('');
  const [duration, setDuration] = useState('');
  const [ethEstimate, setEthEstimate] = useState(null);
  const [txHash, setTxHash] = useState(null);
  const [error, setError] = useState('');

  const publicClient = usePublicClient();
  const { writeContractAsync } = useWriteContract();
  const { isLoading, isSuccess } = useWaitForTransactionReceipt({
    hash: txHash,
  });

  useEffect(() => {
    if (isSuccess) {
      setLandlord('');
      setRentUSD('');
      setDuration('');
      setEthEstimate(null);
      setTxHash(null);
    }
  }, [isSuccess]);

  const handleEstimate = async () => {
    setError('');
    if (!rentUSD) return setError('Enter rent in USD first.');
    try {
      const result = await publicClient.readContract({
        address: CONTRACT_ADDRESS,
        abi: CONTRACT_ABI,
        functionName: 'getETHAmountFromUSD',
        args: [BigInt(parseInt(rentUSD))],
      });
      setEthEstimate(formatEther(result));
    } catch (err) {
      console.error('Estimation error:', err);
      setError('Failed to estimate ETH. Check console for details.');
    }
  };

  const handleCreateRental = async () => {
    setError('');

    if (!landlord || !rentUSD || !duration) {
      return setError('Please fill in all fields.');
    }

    if (!/^0x[a-fA-F0-9]{40}$/.test(landlord)) {
      return setError('Invalid Ethereum address.');
    }

    try {
      const hash = await writeContractAsync({
        address: CONTRACT_ADDRESS,
        abi: CONTRACT_ABI,
        functionName: 'createRental',
        args: [
          landlord,
          BigInt(parseInt(rentUSD)),
          BigInt(parseInt(duration)),
        ],
        value: BigInt(parseInt(Number(ethEstimate) * 1e18)),
      });
      setTxHash(hash);
    } catch (err) {
      console.error('Transaction error:', err);
      setError('Transaction failed. See console.');
    }
  };

  return (
    <div className="bg-white dark:bg-white-900 p-6 rounded-xl shadow-lg w-full max-w-md border border-white-200 dark:border-white-700">
      <h2 className="text-2xl font-bold text-center text-indigo-700 mb-4 text-white">
        Create Rental Agreement
      </h2>

      <div className="flex flex-col gap-3">
        <input
          type="text"
          className="p-2 rounded border"
          placeholder="Landlord Address"
          value={landlord}
          onChange={(e) => setLandlord(e.target.value)}
        />
        <input
          type="number"
          className="p-2 rounded border"
          placeholder="Rent (USD)"
          value={rentUSD}
          onChange={(e) => setRentUSD(e.target.value)}
        />
        <input
          type="number"
          className="p-2 rounded border"
          placeholder="Duration (seconds)"
          value={duration}
          onChange={(e) => setDuration(e.target.value)}
        />
      </div>

      {ethEstimate && (
        <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
          Estimated ETH Required: <span className="font-semibold">{ethEstimate}</span> ETH
        </p>
      )}

      {error && (
        <p className="text-red-600 mt-2 text-sm">{error}</p>
      )}

      <div className="flex flex-col sm:flex-row gap-3 mt-4">
        <button
          onClick={handleEstimate}
          disabled={isLoading}
          className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 disabled:opacity-50"
        >
          {isLoading ? 'Estimating...' : 'Estimate ETH'}
        </button>

        <button
          onClick={handleCreateRental}
          disabled={!ethEstimate || isLoading}
          className="bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700 disabled:opacity-50"
        >
          {isLoading ? 'Submitting...' : 'Submit Rental'}
        </button>
      </div>

      {isLoading && (
        <p className="mt-2 text-yellow-600 text-sm">Waiting for confirmation...</p>
      )}

      {isSuccess && (
        <p className="mt-2 text-green-600 text-sm">✅ Rental created successfully!</p>
      )}
    </div>
  );
}
