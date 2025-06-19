import React, { useEffect, useState } from 'react';
import { usePublicClient } from 'wagmi';
import { CONTRACT_ABI, CONTRACT_ADDRESS } from '../utils/constants';

export default function CurrentRentals() {
  const [rentals, setRentals] = useState([]);
  const [loading, setLoading] = useState(false);
  const publicClient = usePublicClient();

  const fetchActiveRentals = async () => {
    setLoading(true);
    try {
      const result = await publicClient.readContract({
        address: CONTRACT_ADDRESS,
        abi: CONTRACT_ABI,
        functionName: 'getMyRentals', // 🔁 Make sure this exists in your contract
      });
      setRentals(result);
    } catch (err) {
      console.error('Error fetching rentals:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchActiveRentals();
  }, []);

  return (
    <div className="p-4">
      <h2 className="text-lg font-bold mb-2 text-white">Current Rentals</h2>
      {loading ? (
        <p className='text-white'>Loading...</p>
      ) : rentals.length === 0 ? (
        <p className='text-white'>No active rentals found.</p>
      ) : (
        <ul className="space-y-4 text-white">
          {rentals.map((rental, index) => (
            <li key={index} className="p-4 border rounded-md shadow-sm text-white">
              <p><strong>Tenant:</strong> {rental.tenant}</p>
              <p><strong>Landlord:</strong> {rental.landlord}</p>
              <p><strong>Rent:</strong> {Number(rental.rentUSD)} USD</p>
              <p><strong>Ends At:</strong> {new Date((Number(rental.startTime) + Number(rental.duration)) * 1000).toLocaleString()}</p>
            <p className={(Number(rental.startTime) + Number(rental.duration)) * 1000 > Date.now() ? 'text-green-600' : 'text-red-600'}>
            Status: {(Number(rental.startTime) + Number(rental.duration)) * 1000 > Date.now() ? 'Active' : 'Expired'}
            </p>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
