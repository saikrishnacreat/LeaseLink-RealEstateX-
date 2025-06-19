import React, { useEffect, useState } from 'react';
import { usePublicClient } from 'wagmi';
import { CONTRACT_ABI, CONTRACT_ADDRESS } from '../utils/constants';

export default function RentalHistory() {
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(false);
  const publicClient = usePublicClient();

  const fetchHistory = async () => {
    setLoading(true);
    try {
      const result = await publicClient.readContract({
        address: CONTRACT_ADDRESS,
        abi: CONTRACT_ABI,
        functionName: 'getAllRentals', // 🔁 Ensure this function exists
      });
      setHistory(result);
    } catch (err) {
      console.error('Error fetching history:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHistory();
  }, []);

  return (
    <div className="p-4 text-white">
      <h2 className="text-lg font-bold mb-2 text-white">Rental History</h2>
      {loading ? (
        <p>Loading history...</p>
      ) : history.length === 0 ? (
        <p>No past rentals found.</p>
      ) : (
        <ul className="space-y-4 text-white">
          {history.map((entry, index) => (
            <li key={index} className="p-4 border rounded-md shadow-sm text-white">
              <p><strong>Tenant:</strong> {entry.tenant}</p>
              <p><strong>Landlord:</strong> {entry.landlord}</p>
              <p><strong>Rent:</strong> {Number(entry.rentUSD)} USD</p>
              <p><strong>Duration:</strong> {Number(entry.duration)} seconds</p>
              {/* <p><strong>Created At:</strong> {new Date(Number(entry.timestamp) * 1000).toLocaleString()}</p> */}
              
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
