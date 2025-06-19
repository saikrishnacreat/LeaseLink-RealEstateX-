import { useState } from 'react';
import CreateRentalForm from '../components/CreateRentalForm';
import RentalHistory from '../components/RentalHistory';
import CurrentRentals from '../components/CurrentRentals';
import WithdrawButton from '../components/WithdrawRentButton';

export default function Dashboard() {
  const [activeTab, setActiveTab] = useState('create');

  const tabClass = (tab) =>
    `px-4 py-2 rounded-t-md font-medium text-sm transition ${
      tab === activeTab
        ? 'bg-white text-indigo-600 dark:bg-gray-800 dark:text-white'
        : 'bg-gray-100 text-gray-600 hover:text-indigo-600 dark:bg-gray-900 dark:text-gray-400'
    }`;

  return (
    <div className="min-h-screen bg-gray-100 dark:bg-black p-6">
      <div className="max-w-5xl mx-auto">
        <h1 className="text-3xl font-bold text-indigo-700 dark:text-white mb-6 text-center">
          LeaseLink Dashboard
        </h1>

        <div className="flex gap-4 border-b border-white-200 dark:border-white-700 mb-4">
          <button className={tabClass('create')} onClick={() => setActiveTab('create')}>
            Create Rental
          </button>
          <button className={tabClass('history')} onClick={() => setActiveTab('history')}>
            Rental History
          </button>
          <button className={tabClass('active')} onClick={() => setActiveTab('active')}>
            Current Rentals
          </button>
          <button className={tabClass('withdraw')} onClick={() => setActiveTab('withdraw')}>
            Withdraw
          </button>
        </div>

        <div className="bg-white dark:bg-gray-900 rounded-xl shadow-md p-6">
          {activeTab === 'create' && <CreateRentalForm />}
          {activeTab === 'history' && <RentalHistory />}
          {activeTab === 'active' && <CurrentRentals />}
          {activeTab === 'withdraw' && <WithdrawButton />}
        </div>
      </div>
    </div>
  );
}
