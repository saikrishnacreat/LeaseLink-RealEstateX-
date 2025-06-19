import { useState } from 'react';
import { useWriteContract, useWaitForTransactionReceipt } from 'wagmi';
import { CONTRACT_ABI, CONTRACT_ADDRESS } from '../utils/constants';

export default function WithdrawRentButton() {
  const [rentalId, setRentalId] = useState('');
  const [txHash, setTxHash] = useState(null);

  const { writeContractAsync } = useWriteContract();
  const { isLoading, isSuccess, isError } = useWaitForTransactionReceipt({ hash: txHash });

  const handleWithdraw = async () => {
    try {
      const hash = await writeContractAsync({
        address: CONTRACT_ADDRESS,
        abi: CONTRACT_ABI,
        functionName: 'withdrawRent',
        args: [BigInt(rentalId)],
      });
      setTxHash(hash);
    } catch (err) {
      console.error('Withdraw failed:', err);
    }
  };

  return (
    <div className="p-4 border rounded-xl shadow-md w-full max-w-md mt-6">
      <h2 className="text-xl mb-4 font-bold text-white">Withdraw Rent</h2>

      <input
        type="number"
        className="w-full mb-3 p-2 border text-white"
        placeholder="Rental ID"
        value={rentalId}
        onChange={(e) => setRentalId(e.target.value)}
      />

      <button
        onClick={handleWithdraw}
        className="bg-purple-600 text-white px-4 py-2 rounded text-white"
        disabled={!rentalId}
      >
        Withdraw
      </button>

      {isLoading && <p className="mt-2 text-yellow-600">Processing transaction...</p>}
      {isSuccess && <p className="mt-2 text-green-600">✅ Rent withdrawn!</p>}
      {isError && <p className="mt-2 text-red-600">❌ Error withdrawing rent.</p>}
    </div>
  );
}
